package botbase

import (
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"google.golang.org/protobuf/reflect/protoreflect"
	"fmt"
	log "github.com/sirupsen/logrus"
	pb "github.com/jamesread/japella/gen/protobuf"
	"reflect"
	"sync"
)

type Bot struct {
	name string

	logger *log.Logger

	bangCommands map[string]func(*pb.IncomingMessage)
}

func (b *Bot) Start() {
}

func (b *Bot) Stop() {
}

func (b *Bot) SetName(name string) {
	b.name = name
}

func (b *Bot) Name() string {
	if b.name == "" {
		b.name = "Untitled Bot"
	}

	return b.name
}

type PrefixFormatter struct {
	Prefix string
	Formatter log.Formatter
}

func (f *PrefixFormatter) Format(entry *log.Entry) ([]byte, error) {
	entry.Message = fmt.Sprintf("%s %s", f.Prefix, entry.Message)

	f.Formatter.(*log.TextFormatter).DisableTimestamp = true

	return f.Formatter.Format(entry)
}

func (b *Bot) Logger() *log.Logger {
	if b.logger == nil {
		logger := log.New()
		logger.SetFormatter(&PrefixFormatter{
			Prefix: "[Bot: " + b.Name() + "]",
			Formatter: &log.TextFormatter{},
		})

		b.logger = logger
		b.logger.Infof("Logger created for %v", b.Name())
	}

	return b.logger
}

func (b *Bot) RegisterBangCommand(command string, handler func(*pb.IncomingMessage)) {
	b.bangCommands[command] = handler
}

func (b *Bot) Setup() {
	b.bangCommands = make(map[string]func(*pb.IncomingMessage))

	common := runtimeconfig.CommonConfig{}
	runtimeconfig.LoadConfigCommon(&common)
}

func (b *Bot) ConsumeBangCommands() *sync.WaitGroup {
	b.Logger().Infof("ConsumeBangCommands")

	_, handler := amqp.ConsumeForever("IncomingMessage", func(d amqp.Delivery) {
		b.Logger().Infof("consumeBangCommands")

		msg := &pb.IncomingMessage{}

		amqp.Decode(d.Message.Body, &msg)

		b.Logger().Infof("Received %+v", msg)

		if len(msg.Content) > 0 {
			if msg.Content[0] == '!' {
				b.handleBangCommand(msg)
			}
		}
	})

	handler.Wait()

	return handler
}

func (b *Bot) Config() {
}

func (b *Bot) handleBangCommand(msg *pb.IncomingMessage) {
	command := msg.Content[1:]

	b.Logger().Infof("Command: %v", command)

	handler, ok := b.bangCommands[command]

	if ok {
		b.Logger().Infof("Handled command message: %+v", command)
		handler(msg)
	} else {
		b.Logger().Warnf("Unhandled message: %v, %+v", command, msg)

		for k, _ := range b.bangCommands {
			b.Logger().Warnf("Available command: %v", k)
		}
	}
}

type MessageHandler[M interface{}] func(msg M)

func Consume[M interface{}](handler func(M)) {
	log.Infof("Consume")

	var msg M

	messageType := fmt.Sprintf("%+v", reflect.TypeOf(msg).Name())


	log.Infof("Consuming messages of type: %v", messageType)

	_, consumeHandler := amqp.ConsumeForever(messageType, func(d amqp.Delivery) {
		log.Infof("Received message: %v", d)

		err := amqp.Decode(d.Message.Body, &msg)

		if err != nil {
			log.Errorf("Failed to unmarshal message: %v", err)
			return
		}

		handler(msg)
	});

	consumeHandler.Wait()
}

func (b *Bot) SendMessage(msg *pb.OutgoingMessage) {
	amqp.PublishPbWithRoutingKey(msg, msg.Protocol + "-OutgoingMessage")
}

func (b *Bot) Publish(msg protoreflect.ProtoMessage) {
	messageType := fmt.Sprintf("%v", msg.ProtoReflect().Descriptor().FullName())

	amqp.PublishPbWithRoutingKey(msg, messageType)
}
