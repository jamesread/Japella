package discord

import (
	"github.com/bwmarrin/discordgo"
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	log "github.com/sirupsen/logrus"
)

var BotId string
var goBot *discordgo.Session

func Start(appId string, publicKey string, token string) {
	var err error
	goBot, err = discordgo.New("Bot " + token)

	if err != nil {
		log.Errorf("Cannot create new discord bot: %v", err)
		return
	}

	u, err := goBot.User("@me")

	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	BotId = u.ID

	goBot.AddHandler(messageHandler)
	goBot.Identify.Intents = discordgo.IntentsGuildMessages

	err = goBot.Open()

	go Replier()

	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	log.Infof("Discord adaptor bot is running !")
}

func Replier() {
	amqp.ConsumeForever("MessageReply", func(d amqp.Delivery) {
		reply := pb.MessageReply{}

		amqp.Decode(d.Message.Body, &reply)

		log.Infof("reply: %+v %v", reply, goBot)

		goBot.ChannelMessageSend(reply.Channel, reply.Content)

	})

}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Infof("msg2: %+v %v", m.Message, m.Content)

	if m.Author.ID == s.State.User.ID {
		log.Infof("Ignoring msg from myself")
		return
	}

	msg := pb.MessageReceived {
		Author: m.Author.ID,
		Content: m.Content,
		Channel: m.ChannelID,
	}

	amqp.PublishPb(&msg)

	//_, _ = s.ChannelMessageSend(m.ChannelID, "pong")
}
