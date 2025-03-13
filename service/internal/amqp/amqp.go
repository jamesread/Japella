package amqp

import (
	"fmt"
	"github.com/jamesread/japella/internal/runtimeconfig"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
	"os"
	"reflect"
	"sync"
	"time"
)

var (
	conn                 *amqp.Connection
	channels             map[string]*amqp.Channel
	ConnectionIdentifier string
	connMutex            sync.Mutex

	InstanceId string

	AmqpHost string
	AmqpUser string
	AmqpPass string
	AmqpPort int
)

func initAmqp() {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)

	if err != nil {
		log.Fatalf("Could not generate AMQP shortid %v", err)
	}

	InstanceId, _ = sid.Generate()

	cfg := runtimeconfig.Get().Amqp

	AmqpHost = cfg.Host
	AmqpUser = cfg.User
	AmqpPass = cfg.Pass
	AmqpPort = cfg.Port
}

// A dumb Delivery wrapper, so dependencies on this lib don't have to depened on the streadway lib
type Delivery struct {
	Message amqp.Delivery
}

type HandlerFunc func(d Delivery)

func getDialURL() string {
	log.WithFields(log.Fields{
		"host":     AmqpHost,
		"user":     AmqpUser,
		"port":     AmqpPort,
		"instance": InstanceId,
	}).Debugf("AMQP Dial URL")

	return fmt.Sprintf("amqp://%v:%v@%v:%v", AmqpUser, AmqpPass, AmqpHost, AmqpPort)
}

func GetChannel(name string) (*amqp.Channel, error) {
	conn, err := getConn()

	if err != nil {
		return nil, err
	}

	if channel, ok := channels[name]; !ok || channel.IsClosed() {
		log.Debugf("GetChannel() - Opening new channel for %v", name)

		channel, err = conn.Channel()

		if err != nil {
			log.Warnf("GetChannel() - Error opening channel: %v")
			return nil, err
		}

		channels[name] = channel

		declareExchange(channel)
	}

	return channels[name], nil
}

func declareExchange(channel *amqp.Channel) {
	err := channel.ExchangeDeclarePassive(
		"ex_japella",
		"direct",
		true,
		false,
		false,
		false, // nowait
		nil,
	)

	if err != nil {
		log.Errorf("declareExchange() - err: %v. Will also pause.", err)

		time.Sleep(10 * time.Second)
	}
}

func getConn() (*amqp.Connection, error) {
	var err error
	connMutex.Lock()

	if conn == nil || conn.IsClosed() {
		initAmqp()
		log.Infof("AMQP Connecting...")

		cfg := amqp.Config{
			Properties: amqp.Table{
				"connection_name": ConnectionIdentifier + " on " + getHostname(),
			},
		}

		conn, err = amqp.DialConfig(getDialURL(), cfg)

		if err != nil {
			log.Warnf("getConn() - Could not connect: %s", err)
			connMutex.Unlock()

			return nil, err
		} else {
			log.Infof("AMQP Connected")
		}

		channels = make(map[string]*amqp.Channel)
	}

	connMutex.Unlock()

	return conn, nil
}

func getMsgType(msg interface{}) string {
	if t := reflect.TypeOf(msg); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func Publish(routingKey string, msg amqp.Publishing) error {
	channel, err := GetChannel("Publish-" + routingKey)

	if err != nil {
		log.Errorf("Publish error: %v", err)
		return err
	} else {
		return PublishWithChannel(channel, routingKey, msg)
	}
}

func PublishWithChannel(c *amqp.Channel, routingKey string, msg amqp.Publishing) error {
	log.Infof("Publishing to %v", routingKey)

	err := c.Publish(
		"ex_japella",
		routingKey,
		false, // mandatory
		false, // immediate
		msg,
	)

	if err != nil {
		log.Errorf("PublishWithChannel error: %v", err)
	}

	return err
}

func PublishPbWithRoutingKey(msg interface{}, routingKey string) {
	channel, err := GetChannel("Publish-" + getMsgType(msg))

	if err != nil {
		log.Errorf("PublishPbWithRoutingKey: %v", err)
		return
	}

	env := newEnvelope(getMsgType(msg), Encode(msg))

	err = PublishWithChannel(channel, routingKey, env)

	if err != nil {
		log.Errorf("PublishPbWithRoutingKey: %v", err)
	}
}

func PublishPb(msg interface{}) {
	channel, err := GetChannel("Publish-" + getMsgType(msg))

	if err != nil {
		log.Errorf("PublishPb: %v", err)
		return
	}

	PublishPbWithChannel(channel, msg)
}

func PublishPbWithChannel(c *amqp.Channel, msg interface{}) {
	msgType := getMsgType(msg)

	env := newEnvelope(msgType, Encode(msg))

	log.Debugf("PublishPbWithChannel: %+v", msgType)

	err := Publish(msgType, env)

	if err != nil {
		log.Errorf("PublishPbWithChannel: %+v", err)
	}
}

func getHostname() string {
	hostname, err := os.Hostname()

	if err != nil {
		return "unknown"
	}

	return hostname
}

func ConsumeSingle(deliveryTag string, handlerFunc HandlerFunc) *sync.WaitGroup {
	return Consume(deliveryTag, handlerFunc, 1)
}

func ConsumeForever(deliveryTag string, handlerFunc HandlerFunc) *sync.WaitGroup {
	return Consume(deliveryTag, handlerFunc, 0)
}

func Consume(deliveryTag string, handlerFunc HandlerFunc, count int) *sync.WaitGroup {
	handlerDone := &sync.WaitGroup{}
	handlerDone.Add(count)

	if count == 0 {
		handlerDoneInfinate := &sync.WaitGroup{}
		handlerDoneInfinate.Add(1)

		go consumeForever(nil, deliveryTag, handlerFunc)

		return handlerDoneInfinate
	} else {
		go consumeForever(handlerDone, deliveryTag, handlerFunc)

		return handlerDone
	}
}

func consumeForever(handlerDone *sync.WaitGroup, deliveryTag string, handlerFunc HandlerFunc) {
	for {
		channel, err := GetChannel(deliveryTag)

		if err != nil {
			log.Errorf("Get Channel error: %v", err)
		} else {
			consumeWithChannel(handlerDone, channel, deliveryTag, handlerFunc)
		}

		time.Sleep(10 * time.Second)
	}
}

func consumeWithChannelForever(handlerWait *sync.WaitGroup, c *amqp.Channel, deliveryTag string, handlerFunc HandlerFunc) {
	for {
		log.Infof("Consumer channel creating for: %v", deliveryTag)

		consumeWithChannel(handlerWait, c, deliveryTag, handlerFunc)

		log.Infof("Consumer channel closed for: %v", deliveryTag)
		time.Sleep(10 * time.Second)
	}
}

func consumeWithChannel(handlerWait *sync.WaitGroup, c *amqp.Channel, deliveryTag string, handlerFunc HandlerFunc) {
	queueName := "japella-" + getHostname() + "-" + InstanceId + "-" + deliveryTag

	_, err := c.QueueDeclare(
		queueName,
		false, // durable
		true,  // delete when unused
		false, // exclusive
		true,  // nowait
		nil,   // args
	)

	if err != nil {
		log.Warnf("Queue declare error: %v", err)
		return
	}

	err = c.QueueBind(
		queueName,
		deliveryTag, // key
		"ex_japella",
		true, // nowait
		nil,  // args
	)

	if err != nil {
		log.Warnf("Queue bind error: %v %v", deliveryTag, err)
		return
	}

	deliveries, err := c.Consume(
		queueName,              // name
		"consume-"+deliveryTag, // consumer tag
		false,                  // noAck
		false,                  // exclusive
		false,                  // noLocal
		false,                  // noWait
		nil,                    // arguments
	)

	if err != nil {
		log.Warnf("Consumer channel creation error: %v %v", deliveryTag, err)
		return
	}

	consumeDeliveries(deliveries, handlerFunc, handlerWait)

	log.Infof("Consumer deliveries finished: %v", deliveryTag)
}

func newEnvelope(msgType string, body []byte) amqp.Publishing {
	return amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "application/binary",
		Headers: amqp.Table{
			"Upsilon-Msg-Type": msgType,
		},
		Body: body,
	}
}

func consumeDeliveries(deliveries <-chan amqp.Delivery, handlerFunc HandlerFunc, handlerWait *sync.WaitGroup) {
	for d := range deliveries {
		handlerFunc(Delivery{
			Message: d,
		})

		if handlerWait != nil {
			handlerWait.Done()
		}
	}
}

func StartServerListener() {
	log.Info("Started listening")
}
