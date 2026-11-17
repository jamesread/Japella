package discord

import (
	"time"

	"github.com/bwmarrin/discordgo"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/hooks"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
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
		a.failStartup("Failed to initialize bot connection", "Failed to create Discord bot: "+err.Error())
		return nil
	}

	// Store reference to connector instance so messageHandler can access hooks
	discordConnectorInstance = a
	goBot.AddHandler(messageHandler)
	goBot.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	if err = goBot.Open(); err != nil {
		a.Logger().Errorf("Error opening Discord connection: %v", err)
		a.failStartup("Failed to connect to Discord", "Failed to open Discord connection: "+err.Error())
		return nil
	}

	a.resolveBotNickname()
	a.isRunning = true
	a.statusMessage = "Bot is running and connected"
	a.errorMessage = ""
	a.loadHooksFromDB()
	a.registerInteractionHandlers()
	a.Logger().Infof("Discord connector is running!")
	return goBot
}

func (a *DiscordConnector) resolveBotNickname() {
	u, err := goBot.User("@me")
	if err != nil {
		a.Logger().Errorf("Error getting bot user info: %v", err)
		a.nickname = "Discord Bot"
		return
	}
	a.nickname = u.Username
	if a.nickname == "" {
		a.nickname = u.ID
	}
}

func (a *DiscordConnector) loadHooksFromDB() {
	if a.db == nil {
		return
	}
	dbHooks, err := a.db.SelectWebhookHooks("discord", a.botID)
	if err != nil {
		a.Logger().Warnf("Failed to load hooks from database: %v, using config hooks", err)
		return
	}
	if len(dbHooks) > 0 {
		a.hooks = hooks.FromDB(dbHooks)
		a.Logger().Infof("Loaded %d incoming message hook(s) from database", len(a.hooks))
	}
}

func (a *DiscordConnector) registerInteractionHandlers() {
	goBot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := registeredCommands[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	registerCommand("ping", cmdPing)
}

func cmdPing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong!",
		},
	})
	if err != nil {
		log.Warnf("Discord: failed to respond to ping: %v", err)
	}
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
	return &connector.ChatBot{
		BotID:               a.botID,
		Connector:           a.GetProtocol(),
		Name:                discordChatBotName(a.displayName, a.nickname),
		Identity:            a.nickname,
		Icon:                a.GetIcon(),
		IsRunning:           a.isRunning,
		StatusMessage:       chatBotStatusMessage(a.isRunning, a.errorMessage, a.statusMessage),
		ErrorMessage:        a.errorMessage,
		ProtocolDisplayName: "",
	}
}

func discordChatBotName(displayName, nickname string) string {
	if displayName != "" {
		return displayName
	}
	if nickname != "" {
		return nickname
	}
	return "Discord"
}

func chatBotStatusMessage(isRunning bool, errorMessage, statusMessage string) string {
	switch {
	case isRunning:
		return "Bot is running and connected"
	case errorMessage != "":
		return errorMessage
	case statusMessage != "":
		return statusMessage
	default:
		return "Stopped (not connected)"
	}
}

// GetHooks returns the configured hooks for this connector
// Implements ConnectorWithHooks interface
func (a *DiscordConnector) GetHooks() []*connector.IncomingMessageHook {
	hooks := make([]*connector.IncomingMessageHook, 0, len(a.hooks))
	for _, hook := range a.hooks {
		hooks = append(hooks, &connector.IncomingMessageHook{
			URL:           hook.URL,
			Enabled:       hook.Enabled,
			SkipTLSVerify: hook.SkipTLSVerify,
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
			URL:           hook.URL,
			Enabled:       hook.Enabled,
			SkipTLSVerify: hook.SkipTLSVerify,
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

		if err := amqp.Decode(d.Message.Body, &reply); err != nil {
			a.Logger().Errorf("decode OutgoingMessage: %v", err)
			return
		}

		a.Logger().Infof("reply: %+v %v", &reply, goBot)

		// Note: AMQP binding should already filter by identity via routing key, but we keep protocol check for safety

		if goBot != nil {
			if _, err := goBot.ChannelMessageSend(reply.Channel, reply.Content); err != nil {
				a.Logger().Errorf("Failed to send reply to channel %s: %v", reply.Channel, err)
			}

			if a.db != nil {
				conversationKey, conversationTitle := a.db.ResolveOutgoingConversationForLog(
					"discord", a.botID, reply.Channel,
					reply.GetConversationKey(), reply.GetIncommingMessageId(),
				)
				_ = a.db.InsertChatBotMessage(&db.ChatBotMessage{
					Connector:         "discord",
					Identity:          a.nickname,
					BotID:             a.botID,
					ConversationKey:   conversationKey,
					ConversationTitle: conversationTitle,
					Channel:           reply.Channel,
					Author:            "bot",
					Content:           reply.Content,
					Direction:         "outgoing",
					MessageID:         "",
					TimestampUnix:     time.Now().Unix(),
				})
			}
		}

	})
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Infof("discord messageHandler: %+v %v", m.Message, m.Content)

	if m.Author.ID == s.State.User.ID {
		log.Infof("Ignoring msg from myself")
		return
	}

	botIdentity, botUserID := discordBotIdentity()
	if !isBotMentionedAtStart(m, botUserID) {
		log.Infof("Ignoring message that doesn't mention bot at the beginning: Channel=%s, Author=%s, Content=%q", m.ChannelID, getUsername(m), m.Content)
		return
	}

	msg := msgs.IncomingMessage{
		Author:    getUsername(m),
		Content:   removeBotMentionFromStart(m.Content, botUserID),
		Channel:   m.ChannelID,
		MessageId: m.ID,
		Protocol:  "discord",
		Timestamp: time.Now().Unix(),
		Identity:  botIdentity,
	}
	logIncomingDiscordMessage(&msg, botIdentity)
	executeDiscordHooks(&msg)
	amqp.PublishPb(&msg)
}

func discordBotIdentity() (identity, userID string) {
	if goBot == nil || goBot.State == nil || goBot.State.User == nil {
		return "", ""
	}
	identity = goBot.State.User.Username
	userID = goBot.State.User.ID
	if identity == "" {
		identity = goBot.State.User.ID
	}
	return identity, userID
}

func logIncomingDiscordMessage(msg *msgs.IncomingMessage, botIdentity string) {
	if discordConnectorInstance == nil || discordConnectorInstance.db == nil {
		return
	}
	author := msg.Author
	convTitle := author
	if convTitle == "" {
		convTitle = "Unknown sender"
	}
	_ = discordConnectorInstance.db.InsertChatBotMessage(&db.ChatBotMessage{
		Connector:         "discord",
		Identity:          botIdentity,
		BotID:             discordConnectorInstance.botID,
		ConversationKey:   db.BuildConversationKey(msg.Channel, author),
		ConversationTitle: convTitle,
		Channel:           msg.Channel,
		Author:            author,
		Content:           msg.Content,
		Direction:         "incoming",
		MessageID:         msg.MessageId,
		TimestampUnix:     msg.Timestamp,
	})
}

func executeDiscordHooks(msg *msgs.IncomingMessage) {
	if discordConnectorInstance == nil || len(discordConnectorInstance.hooks) == 0 {
		return
	}
	log.Debugf("Executing %d hook(s) for incoming Discord message", len(discordConnectorInstance.hooks))
	hooks.ExecuteHooks(discordConnectorInstance.hooks, msg, discordConnectorInstance.Logger())
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
	if botUserID == "" || m.Content == "" || !mentionsContainBot(m.Mentions, botUserID) {
		return false
	}
	return contentStartsWithBotMention(m.Content, botUserID)
}

func mentionsContainBot(mentions []*discordgo.User, botUserID string) bool {
	for _, mention := range mentions {
		if mention.ID == botUserID {
			return true
		}
	}
	return false
}

func contentStartsWithBotMention(content, botUserID string) bool {
	for _, pattern := range botMentionPatterns(botUserID) {
		if !hasPrefixExact(content, pattern) {
			continue
		}
		if len(content) == len(pattern) || content[len(pattern)] == ' ' {
			return true
		}
	}
	return false
}

func botMentionPatterns(botUserID string) []string {
	return []string{
		"<@" + botUserID + ">",
		"<@!" + botUserID + ">",
	}
}

func hasPrefixExact(content, pattern string) bool {
	return len(content) >= len(pattern) && content[:len(pattern)] == pattern
}

// removeBotMentionFromStart removes the bot mention from the beginning of the message content
func removeBotMentionFromStart(content string, botUserID string) string {
	if botUserID == "" {
		return content
	}
	for _, pattern := range botMentionPatterns(botUserID) {
		if remaining, ok := stripExactPrefix(content, pattern); ok {
			return remaining
		}
	}
	return content
}

func stripExactPrefix(content, pattern string) (string, bool) {
	if !hasPrefixExact(content, pattern) {
		return content, false
	}
	remaining := content[len(pattern):]
	if len(remaining) > 0 && remaining[0] == ' ' {
		remaining = remaining[1:]
	}
	return remaining, true
}
