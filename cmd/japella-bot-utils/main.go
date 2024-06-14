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
	_, handler := amqp.ConsumeForever("MessageReceived", func(d amqp.Delivery) {
		msg := &pb.MessageReceived{}

		amqp.Decode(d.Message.Body, &msg)

		log.Infof("recv: %+v", msg)

		handleMessage(msg)
	})

	handler.Wait()
	log.Infof("done")
}

func handleMessage(msg *pb.MessageReceived) {
	switch msg.Content {
	case "!test":
		replyTest(msg)
		break
	default:
		log.Warnf("Unhandled message: %+v", msg)
	}
}

func replyTest(msg *pb.MessageReceived) {
	reply := &pb.MessageReply{
		Channel: msg.Channel,
		Content: "This is a reply",
	}

	amqp.PublishPb(reply)
}
