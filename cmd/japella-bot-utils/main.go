package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/amqp"
	pb "github.com/jamesread/japella/gen/protobuf"
)

var cfg struct {
	Common *runtimeconfig.CommonConfig
}

func main() {
	log.Infof("japella-bot-utils")

	runtimeconfig.LoadConfigCommon(cfg.Common);

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
