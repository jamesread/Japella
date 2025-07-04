package discord

import (
	"github.com/bwmarrin/discordgo"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	log "github.com/sirupsen/logrus"
	"time"
)

var goBot *discordgo.Session
var registeredCommands map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)

func (a *DiscordConnector) startActual(token string) *discordgo.Session {
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

	a.nickname = u.ID

	goBot.AddHandler(messageHandler)
	goBot.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = goBot.Open()

	if err != nil {
		log.Errorf("err: %v", err)
	}

	goBot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := registeredCommands[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	registerCommand("ping", cmdPing)

	if false {
		go Replier()
		go MessageSearch()
	}

	if err != nil {
		log.Fatalf("%v", err)
		return nil
	}

	a.Logger().Infof("Connector is running !")

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
		Name:        name,
		Description: "A japella command",
	})

	if err != nil {
		log.Errorf("register cmd err: %v", err)
	}

	registeredCommands[name] = handler
}

func MessageSearch() {
	log.Info("msg search")
	amqp.ConsumeForever("ThreadSearchRequest", func(d amqp.Delivery) {
		log.Infof("searching for messages")

		/*
			for _, guild := range goBot.State.Guilds {
				log.Infof("guild: %v %v", guild.ID, guild.Name)

				channels, err := goBot.GuildChannels(guild.ID)

				if err != nil {
					log.Errorf("channels err: %v", err)
					continue
				}

				for _, channel := range channels {
					log.Infof("channel: %v %v %v", channel.ID, channel.Name, channel.Type)
				}
			}
		*/

		res, err := goBot.GuildThreadsActive(
			"846737624960860180",
		)

		if err != nil {
			log.Errorf("threads err: %v", err)
		}

		for _, thread := range res.Threads {
			log.Infof("res: %v %v %v", thread.ID, thread.Name, thread.ParentID)

			lastMsg, err := goBot.ChannelMessage(thread.ParentID, thread.LastMessageID)

			if err != nil {
				log.Errorf("msg err: %v %v %v", err, thread.ParentID, thread.LastMessageID)
				continue
			}

			log.Infof("msg: %v", lastMsg)
		}

		msg := &msgs.ThreadSearchResponse{}

		amqp.PublishPbWithRoutingKey(msg, "ThreadSearchResponse")
		//		amqp.PublishPb(msg)
	})
}

func (a *DiscordConnector) GetIdentity() string {
	return a.nickname
}

func (a *DiscordConnector) GetProtocol() string {
	return "discord"
}

func Replier() {
	amqp.ConsumeForever("discord-OutgoingMessage", func(d amqp.Delivery) {
		reply := msgs.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		log.Infof("reply: %+v %v", &reply, goBot)

		goBot.ChannelMessageSend(reply.Channel, reply.Content)

	})

}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Infof("discord messageHandler: %+v %v", m.Message, m.Content)

	if m.Author.ID == s.State.User.ID {
		log.Infof("Ignoring msg from myself")
		return
	}

	msg := msgs.IncomingMessage{
		Author:    getUsername(m),
		Content:   m.Content,
		Channel:   m.ChannelID,
		MessageId: m.ID,
		Protocol:  "discord",
		Timestamp: time.Now().Unix(),
	}

	amqp.PublishPb(&msg)

	// _, _ = s.ChannelMessageSend(m.ChannelID, "pong")
}

func getUsername(m *discordgo.MessageCreate) string {
	if m.Member != nil {
		if m.Member.Nick != "" {
			return m.Member.Nick
		}

		if m.Member.User != nil {
			return m.Member.User.GlobalName
		}
	}

	return m.Author.Username
}
