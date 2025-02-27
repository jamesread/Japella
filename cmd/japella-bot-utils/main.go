package main

import (
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

type LocalConfig struct {
	Common    *runtimeconfig.CommonConfig
	AppId     string
	PublicKey string
	Token     string
}

func main() {
	log.Infof("japella-bot-utils")

	cfg := &LocalConfig{}
	cfg.Common = runtimeconfig.LoadNewConfigCommon()

	log.Infof("cfg: %+v", cfg)

	Start()
}

func Start() {
	_, handler := amqp.ConsumeForever("IncomingMessage", func(d amqp.Delivery) {
		msg := &pb.IncomingMessage{}

		amqp.Decode(d.Message.Body, &msg)

		log.Infof("recv: %+v", msg)

		handleMessage(msg)
	})

	handler.Wait()
	log.Infof("done")
}

func handleMessage(msg *pb.IncomingMessage) {
	switch msg.Content {
	case "!test":
		replyTest(msg)
		break
	default:
		log.Warnf("Unhandled message: %+v", msg)
	}
}

func replyTest(msg *pb.IncomingMessage) {
	reply := &pb.OutgoingMessage{
		Channel: msg.Channel,
		Content: "This is a reply",
		IncomingMessageId: msg.MessageId,
		Protocol: msg.Protocol,
	}

	amqp.PublishPbWithRoutingKey(reply, msg.Protocol + "-OutgoingMessage")
}
