package discord

import (
	"github.com/bwmarrin/discordgo"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/hooks"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
	"time"
)

var goBot *discordgo.Session
var registeredCommands map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
var discordConnectorInstance *DiscordConnector // Global reference to access hooks from messageHandler

func (a *DiscordConnector) startActual(token string) *discordgo.Session {
	var err error
	goBot, err = discordgo.New("Bot " + token)

	registeredCommands = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

	if err != nil {
		a.Logger().Errorf("Cannot create new discord bot: %v", err)
		a.isRunning = false
		return nil
	}

	// Store reference to connector instance so messageHandler can access hooks
	discordConnectorInstance = a

	goBot.AddHandler(messageHandler)
	goBot.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = goBot.Open()

	if err != nil {
		a.Logger().Errorf("Error opening Discord connection: %v", err)
		a.isRunning = false
		return nil
	}

	// Get user info after connection is open
	u, err := goBot.User("@me")
	if err != nil {
		a.Logger().Errorf("Error getting bot user info: %v", err)
		// Don't fail completely, just use a default
		a.nickname = "Discord Bot"
	} else {
		a.nickname = u.Username
		if a.nickname == "" {
			a.nickname = u.ID
		}
	}

	a.isRunning = true

	// Load hooks from database now that we have the nickname
	if a.db != nil {
		dbHooks, err := a.db.SelectWebhookHooks("discord", a.nickname)
		if err == nil && len(dbHooks) > 0 {
			a.hooks = make([]runtimeconfig.IncomingMessageHook, 0, len(dbHooks))
			for _, hook := range dbHooks {
				a.hooks = append(a.hooks, runtimeconfig.IncomingMessageHook{
					URL:     hook.URL,
					Enabled: hook.Enabled,
				})
			}
			a.Logger().Infof("Loaded %d incoming message hook(s) from database", len(a.hooks))
		} else if err != nil {
			a.Logger().Warnf("Failed to load hooks from database: %v, using config hooks", err)
		}
	}

	goBot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := registeredCommands[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	registerCommand("ping", cmdPing)

	a.Logger().Infof("Discord connector is running!")

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

func (a *DiscordConnector) GetIcon() string {
	return "mdi:discord"
}

func (a *DiscordConnector) GetChatBot() *connector.ChatBot {
	name := "Discord"
	if a.nickname != "" {
		name = a.nickname
	}
	return &connector.ChatBot{
		Connector:          a.GetProtocol(),
		Name:               name,
		Identity:           a.nickname,
		Icon:               a.GetIcon(),
		IsRunning:          a.isRunning,
		ProtocolDisplayName: "", // Discord doesn't have a protocol-specific display name
	}
}

// GetHooks returns the configured hooks for this connector
// Implements ConnectorWithHooks interface
func (a *DiscordConnector) GetHooks() []*connector.IncomingMessageHook {
	hooks := make([]*connector.IncomingMessageHook, 0, len(a.hooks))
	for _, hook := range a.hooks {
		hooks = append(hooks, &connector.IncomingMessageHook{
			URL:     hook.URL,
			Enabled: hook.Enabled,
		})
	}
	return hooks
}

// SetHooks updates the hooks for this connector
// Implements ConnectorWithHooks interface
// Note: This updates in-memory hooks only. The database should be updated via the API.
func (a *DiscordConnector) SetHooks(hooks []*connector.IncomingMessageHook) error {
	a.hooks = make([]runtimeconfig.IncomingMessageHook, 0, len(hooks))
	for _, hook := range hooks {
		a.hooks = append(a.hooks, runtimeconfig.IncomingMessageHook{
			URL:     hook.URL,
			Enabled: hook.Enabled,
		})
	}
	a.Logger().Infof("Updated hooks: %d hook(s) configured", len(a.hooks))
	return nil
}

func (a *DiscordConnector) Replier() {
	// Use identity-specific routing key so this bot only receives messages intended for it
	routingKey := amqp.GetOutgoingMessageRoutingKey("discord", a.nickname)
	a.Logger().Infof("Starting Replier for routing key: %s", routingKey)

	// Consume from identity-specific routing key
	amqp.ConsumeForever(routingKey, func(d amqp.Delivery) {
		reply := msgs.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		a.Logger().Infof("reply: %+v %v", &reply, goBot)

		// Note: AMQP binding should already filter by identity via routing key, but we keep protocol check for safety

		if goBot != nil {
			goBot.ChannelMessageSend(reply.Channel, reply.Content)
		}

	})
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Infof("discord messageHandler: %+v %v", m.Message, m.Content)

	if m.Author.ID == s.State.User.ID {
		log.Infof("Ignoring msg from myself")
		return
	}

	// Get bot identity from the session
	botIdentity := ""
	botUserID := ""
	if goBot != nil && goBot.State != nil && goBot.State.User != nil {
		botIdentity = goBot.State.User.Username
		botUserID = goBot.State.User.ID
		if botIdentity == "" {
			botIdentity = goBot.State.User.ID
		}
	}

	// Only process messages that mention the bot at the beginning
	if !isBotMentionedAtStart(m, botUserID) {
		log.Infof("Ignoring message that doesn't mention bot at the beginning: Channel=%s, Author=%s, Content=%q", m.ChannelID, getUsername(m), m.Content)
		return
	}

	// Remove the bot mention from the beginning of the content for processing
	content := removeBotMentionFromStart(m.Content, botUserID)

	msg := msgs.IncomingMessage{
		Author:    getUsername(m),
		Content:   content, // Use content with bot mention removed
		Channel:   m.ChannelID,
		MessageId: m.ID,
		Protocol:  "discord",
		Timestamp: time.Now().Unix(),
		Identity:  botIdentity, // Include bot identity so hooks can route responses correctly
	}

	// Execute hooks if configured
	if discordConnectorInstance != nil && len(discordConnectorInstance.hooks) > 0 {
		log.Debugf("Executing %d hook(s) for incoming Discord message", len(discordConnectorInstance.hooks))
		hooks.ExecuteHooks(discordConnectorInstance.hooks, &msg, discordConnectorInstance.Logger())
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

// isBotMentionedAtStart checks if the bot is mentioned at the beginning of the message
// Discord mentions can be in the format: @username, <@userID>, or <@!userID>
func isBotMentionedAtStart(m *discordgo.MessageCreate, botUserID string) bool {
	if botUserID == "" {
		return false
	}

	content := m.Content
	if content == "" {
		return false
	}

	// First check if the bot is in the Mentions list (most reliable)
	botMentioned := false
	for _, mention := range m.Mentions {
		if mention.ID == botUserID {
			botMentioned = true
			break
		}
	}

	if !botMentioned {
		return false
	}

	// Check if the mention is at the beginning of the message
	// Discord mentions can be: <@userID> or <@!userID>
	mentionPatterns := []string{
		"<@" + botUserID + ">",
		"<@!" + botUserID + ">",
	}

	// Check if content starts with any mention pattern
	for _, pattern := range mentionPatterns {
		if len(content) >= len(pattern) && content[:len(pattern)] == pattern {
			// Check if it's at the start (possibly followed by space or end of string)
			if len(content) == len(pattern) || content[len(pattern)] == ' ' {
				return true
			}
		}
	}

	return false
}

// removeBotMentionFromStart removes the bot mention from the beginning of the message content
func removeBotMentionFromStart(content string, botUserID string) string {
	if botUserID == "" {
		return content
	}

	mentionPatterns := []string{
		"<@" + botUserID + ">",
		"<@!" + botUserID + ">",
	}

	for _, pattern := range mentionPatterns {
		if len(content) >= len(pattern) && content[:len(pattern)] == pattern {
			// Remove the mention and any following space
			remaining := content[len(pattern):]
			if len(remaining) > 0 && remaining[0] == ' ' {
				remaining = remaining[1:]
			}
			return remaining
		}
	}

	return content
}
