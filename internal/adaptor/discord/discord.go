package discord

import (
	"github.com/bwmarrin/discordgo"
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	log "github.com/sirupsen/logrus"
	"time"
)

var BotId string
var goBot *discordgo.Session
var registeredCommands map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)

func Start(appId string, publicKey string, token string) *discordgo.Session {
	var err error
	goBot, err = discordgo.New("Bot " + token)

	registeredCommands = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

	if err != nil {
		log.Errorf("Cannot create new discord bot: %v", err)
		return nil
	}

	u, err := goBot.User("@me")

	if err != nil {
		log.Fatalf("%v", err)
		return nil
	}

	BotId = u.ID

	goBot.AddHandler(messageHandler)
	goBot.Identify.Intents = discordgo.IntentsGuildMessages

	err = goBot.Open()

	log.Errorf("err: %v", err)

	goBot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := registeredCommands[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	registerCommand("ping", cmdPing);

	go Replier()

	if err != nil {
		log.Fatalf("%v", err)
		return nil
	}

	log.Infof("Discord adaptor bot is running !")

	return goBot
}

func cmdPing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong!",
		},
	})
}

func registerCommand(name string, handler func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	_, err := goBot.ApplicationCommandCreate(goBot.State.User.ID, "", &discordgo.ApplicationCommand{
		Name: name,
		Description: "A japella command",
	})

	log.Errorf("err: %v", err)

	registeredCommands[name] = handler
}

func Replier() {
	amqp.ConsumeForever("discord-OutgoingMessage", func(d amqp.Delivery) {
		reply := pb.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		log.Infof("reply: %+v %v", &reply, goBot)

		goBot.ChannelMessageSend(reply.Channel, reply.Content)

	})

}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Infof("msg2: %+v %v", m.Message, m.Content)

	if m.Author.ID == s.State.User.ID {
		log.Infof("Ignoring msg from myself")
		return
	}

	msg := pb.IncommingMessage{
		Author:  m.Author.ID,
		Content: m.Content,
		Channel: m.ChannelID,
		MessageId: m.ID,
		Protocol: "discord",
		Timestamp: time.Now().Unix(),
	}

	amqp.PublishPb(&msg)

	// _, _ = s.ChannelMessageSend(m.ChannelID, "pong")
}
