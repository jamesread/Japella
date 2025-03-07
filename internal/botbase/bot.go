package botbase

import (
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
	"fmt"
	pb "github.com/jamesread/japella/gen/protobuf"
	"sync"
	"regexp"
)

type Bot struct {
	name string

	utils.LogComponent

	bangCommands map[string]func(*pb.IncomingMessage, string, string)
}

func (b *Bot) Start() {
}

func (b *Bot) Stop() {
}

func (b *Bot) SetName(name string) {
	b.name = name

	b.SetPrefix("Bot: " + b.Name())
}

func (b *Bot) Name() string {
	if b.name == "" {
		b.name = "Untitled Bot"
	}

	return b.name
}

func (b *Bot) RegisterBangCommand(command string, handler func(*pb.IncomingMessage, string, string)) {
	b.bangCommands[command] = handler
}

func (b *Bot) Setup() {
	b.bangCommands = make(map[string]func(*pb.IncomingMessage, string, string))
}

func (b *Bot) ConsumeBangCommands() *sync.WaitGroup {
	b.Logger().Infof("ConsumeBangCommands")

	handler := amqp.ConsumeForever("IncomingMessage", func(d amqp.Delivery) {
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

var commandMatcher = regexp.MustCompile(`^!(\w+)(?:\s+(.*))?$`)

func (b *Bot) handleBangCommand(msg *pb.IncomingMessage) {
	match := commandMatcher.FindStringSubmatch(msg.Content)

	if len(match) < 2 {
		b.Logger().Warnf("Failed to match command: %v", msg.Content)
	} else {
		command := match[1]
		args := match[2]

		b.Logger().Infof("Command: %v", command)

		handler, ok := b.bangCommands[command]

		if ok {
			b.Logger().Infof("Handled command message: %+v", command)
			handler(msg, command, args)
		} else {
			b.Logger().Warnf("Unhandled message: %v, %+v", command, msg)

			for k, _ := range b.bangCommands {
				b.Logger().Warnf("Available command: %v", k)
			}
		}
	}
}

/**
type MessageHandler[M interface{}] func(msg M)

func Consumek[M interface{}](log *utils.LogComponent, handler func(M)) {
	log.Logger().Infof("Consume")

	var msg M

	messageType := fmt.Sprintf("%+v", reflect.TypeOf(msg).Name())


	log.Logger().Infof("Consuming messages of type: %v", messageType)

	consumeHandler := amqp.ConsumeForever(messageType, func(d amqp.Delivery) {
		log.Logger().Infof("Received message: %v", d)

		err := amqp.Decode(d.Message.Body, &msg)

		if err != nil {
			log.Logger().Errorf("Failed to unmarshal message: %v", err)
			return
		}

		handler(msg)
	});

	consumeHandler.Wait()
}
*/

func (b *Bot) SendMessage(msg *pb.OutgoingMessage) {
	amqp.PublishPbWithRoutingKey(msg, msg.Protocol + "-OutgoingMessage")
}

func (b *Bot) Reply(msg *pb.IncomingMessage) *pb.OutgoingMessage {
	return &pb.OutgoingMessage{
		Protocol: msg.Protocol,
		Channel: msg.Channel,
	}
}

func (b *Bot) Publish(msg protoreflect.ProtoMessage) {
	messageType := fmt.Sprintf("%v", msg.ProtoReflect().Descriptor().FullName())

	amqp.PublishPbWithRoutingKey(msg, messageType)
}
