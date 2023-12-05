package amqp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
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

func init() {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)

	if err != nil {
		log.Fatalf("Could not generate AMQP shortid %v", err)
	}

	InstanceId, _ = sid.Generate()
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
	}).Infof("AMQP Dial URL")

	return fmt.Sprintf("amqp://%v:%v@%v:%v", AmqpUser, AmqpPass, AmqpHost, AmqpPort)
}

func GetChannel(name string) (*amqp.Channel, error) {
	conn, err := getConn()

	if err != nil {
		return nil, err
	}

	if channel, ok := channels[name]; !ok {
		log.Debugf("GetChannel() - Opening new channel for %v", name)

		channel, err = conn.Channel()

		if err != nil {
			log.Warnf("GetChannel() - Error opening channel: %v")
			return nil, err
		}

		channels[name] = channel
	}

	return channels[name], nil
}

func getConn() (*amqp.Connection, error) {
	var err error
	connMutex.Lock()

	if conn == nil || conn.IsClosed() {
		log.Debugf("getConn() - Creating conn")

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
	err := c.Publish(
		"ex_japella",
		routingKey,
		false, // mandatory
		false, // immediate
		msg,
	)

	return err
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

func ConsumeSingle(deliveryTag string, handlerFunc HandlerFunc) (*sync.WaitGroup, *sync.WaitGroup) {
	return Consume(deliveryTag, handlerFunc, 1)
}

func ConsumeForever(deliveryTag string, handlerFunc HandlerFunc) (*sync.WaitGroup, *sync.WaitGroup) {
	return Consume(deliveryTag, handlerFunc, 0)
}

func Consume(deliveryTag string, handlerFunc HandlerFunc, count int) (*sync.WaitGroup, *sync.WaitGroup) {
	chanReady := &sync.WaitGroup{}
	chanReady.Add(1)

	handlerDone := &sync.WaitGroup{}
	handlerDone.Add(count)

	if count == 0 {
		handlerDoneInfinate := &sync.WaitGroup{}
		handlerDoneInfinate.Add(1)

		go consumeForever(chanReady, nil, deliveryTag, handlerFunc)

		return chanReady, handlerDoneInfinate
	} else {
		go consumeForever(chanReady, handlerDone, deliveryTag, handlerFunc)

		return chanReady, handlerDone
	}
}

func consumeForever(consumerReady *sync.WaitGroup, handlerDone *sync.WaitGroup, deliveryTag string, handlerFunc HandlerFunc) {
	for {
		channel, err := GetChannel(deliveryTag)

		if err != nil {
			log.Errorf("Get Channel error: %v", err)
		} else {
			consumeWithChannel(consumerReady, handlerDone, channel, deliveryTag, handlerFunc)
		}

		time.Sleep(10 * time.Second)
	}
}

func consumeWithChannel(consumerReady *sync.WaitGroup, handlerWait *sync.WaitGroup, c *amqp.Channel, deliveryTag string, handlerFunc HandlerFunc) {
	queueName := getHostname() + "-" + InstanceId + "-" + deliveryTag

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

	log.Infof("Consumer channel creating for: %v", deliveryTag)

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

	consumerReady.Done()

	consumeDeliveries(deliveries, handlerFunc, handlerWait)

	log.Infof("Consumer channel closed for: %v", deliveryTag)
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
