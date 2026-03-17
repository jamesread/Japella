package telegram

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram/bot"
	tgbotmdl "github.com/go-telegram/bot/models"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/hooks"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
)

type TelegramChannel struct {
	ID       int64  // Chat ID
	Title    string // Channel/chat title
	Type     string // "channel", "group", "supergroup", "private"
	Username string // Username if available
}

type TelegramConnector struct {
	nickname            string
	displayName         string
	protocolDisplayName string // FirstName from GetMe response
	isRunning           bool
	statusMessage       string
	errorMessage        string
	channels            map[int64]*TelegramChannel // Track known channels/chats
	botInstance         *tgbotapi.Bot              // Store bot instance for API calls
	hooks               []runtimeconfig.IncomingMessageHook // Webhooks to call when messages are received
	db                  *db.DB                     // Database reference for loading hooks
	utils.LogComponent

	connector.BaseConnector
	connector.ConnectorWithYamlConfig
}

func (c *TelegramConnector) GetIdentity() string {
	return c.nickname
}

func (c *TelegramConnector) GetProtocol() string {
	return "telegram"
}

func (c *TelegramConnector) GetIcon() string {
	return "mdi:telegram"
}

func (c *TelegramConnector) GetDisplayName() string {
	if c.displayName != "" {
		return c.displayName
	}
	return c.GetProtocol()
}

func (c *TelegramConnector) GetChatBot() *connector.ChatBot {
	name := c.GetDisplayName()
	if name == c.GetProtocol() && c.nickname != "" {
		// If no custom display name is set, use the bot's username
		name = c.nickname
	}

	statusMsg := ""
	if c.isRunning {
		statusMsg = "Bot is running and connected"
	} else if c.errorMessage != "" {
		statusMsg = c.errorMessage
	} else if c.statusMessage != "" {
		statusMsg = c.statusMessage
	} else {
		statusMsg = "Bot is not running"
	}

	return &connector.ChatBot{
		Connector:          c.GetProtocol(),
		Name:               name,
		Identity:           c.nickname,
		Icon:               c.GetIcon(),
		IsRunning:          c.isRunning,
		StatusMessage:      statusMsg,
		ErrorMessage:       c.errorMessage,
		ProtocolDisplayName: c.protocolDisplayName,
	}
}

func (c *TelegramConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	config, _ := startup.Config.(*runtimeconfig.TelegramConfig)

	if config == nil || config.Token == "" {
		c.Logger().Errorf("Telegram bot token is not set in configuration")
		c.statusMessage = "Configuration error: Bot token is not set"
		c.errorMessage = "Bot token is missing from configuration"
		c.isRunning = false
		return
	}

	if config.Name != "" {
		c.displayName = config.Name
	}

	// Store DB reference for later hook loading (after bot starts and nickname is available)
	c.db = startup.DB
	c.hooks = config.IncomingMessageHooks
	if len(c.hooks) > 0 {
		c.Logger().Infof("Configured %d incoming message hook(s) from config (will load from DB after bot starts)", len(c.hooks))
	}

	c.StartWithToken(config.Token)
}

func (c *TelegramConnector) Start() {
	// Bot is started in SetStartupConfiguration via StartWithToken
	// This method exists to satisfy the BaseConnector interface
}

func (c *TelegramConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	// Telegram uses bot tokens, not OAuth, so no refresh is needed
	return nil
}

func (c *TelegramConnector) StartWithToken(token string) {
	c.SetPrefix("Telegram-new")
	c.Logger().Infof("japella-bot-telegram")

	c.startBot(token)
}

func (c *TelegramConnector) startBot(botToken string) {
	c.Logger().Infof("Starting Telegram bot with token (length: %d)", len(botToken))

	// Initialize channels map
	if c.channels == nil {
		c.channels = make(map[int64]*TelegramChannel)
	}

	// Create a handler wrapper to add debugging
	handlerWrapper := func(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
		c.Logger().Infof("=== MESSAGE HANDLER CALLED ===")
		c.Logger().Infof("Update type: Message=%v, CallbackQuery=%v, ChannelPost=%v",
			update.Message != nil,
			update.CallbackQuery != nil,
			update.ChannelPost != nil)

		if update.Message != nil {
			text := ""
			if update.Message.Text != "" {
				text = update.Message.Text
			}
			c.Logger().Infof("Message details - ChatID: %d, From: %+v, Text: %s",
				update.Message.Chat.ID,
				update.Message.From,
				text)
		}

		// Call the actual handler
		c.messageHandler(ctx, b, update)
	}

	opts := []tgbotapi.Option{
		tgbotapi.WithDefaultHandler(handlerWrapper),
	}

	c.Logger().Infof("Creating bot instance with %d options", len(opts))

	var err error

	c.botInstance, err = tgbotapi.New(botToken, opts...)

	if err != nil {
		c.Logger().Errorf("Error creating bot: %v", err)
		c.isRunning = false
		c.errorMessage = "Failed to create bot instance: " + err.Error()
		c.statusMessage = "Failed to initialize bot connection"
		return
	}

	c.Logger().Infof("Bot instance created successfully")

	// Use a separate context for GetMe to avoid cancellation issues
	getMeCtx, getMeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer getMeCancel()

	c.Logger().Infof("Calling GetMe to verify bot token")
	me, err := c.botInstance.GetMe(getMeCtx)
	if err != nil {
		c.Logger().Errorf("Error getting bot info: %v", err)
		c.isRunning = false
		c.errorMessage = "Failed to verify bot token: " + err.Error()
		c.statusMessage = "Bot token validation failed"
		return
	}

	c.nickname = me.Username
	c.protocolDisplayName = me.FirstName // Store FirstName as protocol display name
	c.isRunning = true
	c.statusMessage = "Bot is running and connected"
	c.errorMessage = "" // Clear any previous error messages

	c.SetPrefix("Telegram-" + c.nickname)
	c.Logger().Infof("Bot verified successfully - Username: %s, ID: %d, FirstName: %s, CanJoinGroups: %v, CanReadAllGroupMessages: %v",
		me.Username, me.ID, me.FirstName, me.CanJoinGroups, me.CanReadAllGroupMessages)

	// Start AMQP replier after nickname is set so it can bind to the correct identity-specific routing key
	if runtimeconfig.Get().Amqp.Enabled {
		c.Logger().Infof("Starting AMQP replier with identity: %s", c.nickname)
		go c.Replier()
	}

	// Load hooks from database now that we have the nickname
	if c.db != nil {
		dbHooks, err := c.db.SelectWebhookHooks("telegram", c.nickname)
		if err == nil && len(dbHooks) > 0 {
			c.hooks = make([]runtimeconfig.IncomingMessageHook, 0, len(dbHooks))
			for _, hook := range dbHooks {
				c.hooks = append(c.hooks, runtimeconfig.IncomingMessageHook{
					URL:     hook.URL,
					Enabled: hook.Enabled,
				})
			}
			c.Logger().Infof("Loaded %d incoming message hook(s) from database", len(c.hooks))
		} else if err != nil {
			c.Logger().Warnf("Failed to load hooks from database: %v, using config hooks", err)
		}
	}

	// Send debug message if configured
	debugChatId := runtimeconfig.Get().TelegramDebugChatId
	if debugChatId != "" {
		debugCtx, debugCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer debugCancel()
		c.sendDebugStartupMessage(debugCtx, debugChatId, me.Username)
	}

	// Create a context for Start that will be cancelled on interrupt
	startCtx, startCancel := signal.NotifyContext(context.Background(), os.Interrupt)

	c.Logger().Infof("Starting bot polling in goroutine...")
	go func() {
		defer startCancel()
		c.Logger().Infof("Bot.Start() called - beginning to poll for updates")
		c.botInstance.Start(startCtx)
		c.Logger().Infof("Bot.Start() returned (context cancelled)")
		c.isRunning = false
		c.statusMessage = "Bot polling stopped"
	}()

	c.Logger().Infof("Bot startup complete - isRunning: %v", c.isRunning)
}

// GetBotInstance returns the bot instance (for internal use)
func (c *TelegramConnector) GetBotInstance() *tgbotapi.Bot {
	return c.botInstance
}

// sendDebugStartupMessage sends a startup message to the debug chat ID if configured
func (c *TelegramConnector) sendDebugStartupMessage(ctx context.Context, chatIdStr string, botUsername string) {
	chatId, err := strconv.ParseInt(chatIdStr, 10, 64)
	if err != nil {
		c.Logger().Errorf("Invalid telegramDebugChatId format (must be numeric): %s, error: %v", chatIdStr, err)
		return
	}

	message := fmt.Sprintf("🤖 Bot started successfully!\n\nUsername: @%s\nStatus: Running and ready to receive messages", botUsername)

	c.Logger().Infof("Sending debug startup message to chat ID: %d", chatId)

	_, err = c.botInstance.SendMessage(ctx, &tgbotapi.SendMessageParams{
		ChatID: chatId,
		Text:   message,
	})

	if err != nil {
		c.Logger().Errorf("Failed to send debug startup message to chat ID %d: %v", chatId, err)
		c.Logger().Warnf("This might mean the bot doesn't have access to the chat, or the chat ID is incorrect")
	} else {
		c.Logger().Infof("Debug startup message sent successfully to chat ID: %d", chatId)
	}
}

func (c *TelegramConnector) messageHandler(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
	c.Logger().Infof("=== messageHandler ENTERED ===")

	// Log all update types
	if update.Message != nil {
		c.Logger().Infof("Processing Message update")
		c.handleMessage(update.Message)
	} else if update.ChannelPost != nil {
		c.Logger().Infof("Processing ChannelPost update")
		c.handleMessage(update.ChannelPost)
	} else if update.EditedMessage != nil {
		c.Logger().Infof("Processing EditedMessage update")
		c.handleMessage(update.EditedMessage)
	} else if update.CallbackQuery != nil {
		c.Logger().Infof("Processing CallbackQuery update (ID: %s)", update.CallbackQuery.ID)
	} else if update.InlineQuery != nil {
		c.Logger().Infof("Processing InlineQuery update (ID: %s)", update.InlineQuery.ID)
	} else {
		c.Logger().Warnf("Received update with unknown type")
		c.Logger().Infof("Full update structure: %+v", update)
	}

	c.Logger().Infof("=== messageHandler EXITING ===")
}

func (c *TelegramConnector) handleMessage(message *tgbotmdl.Message) {
	c.Logger().Infof("handleMessage called - ChatID: %d", message.Chat.ID)

	// Track the channel/chat
	chat := message.Chat
	channel := &TelegramChannel{
		ID:       chat.ID,
		Title:    chat.Title,
		Type:     string(chat.Type),
		Username: chat.Username,
	}
	// If title is empty, try to get it from the chat
	if channel.Title == "" && chat.FirstName != "" {
		channel.Title = chat.FirstName
		if chat.LastName != "" {
			channel.Title += " " + chat.LastName
		}
	}
	if channel.Title == "" && channel.Username != "" {
		channel.Title = "@" + channel.Username
	}
	if channel.Title == "" {
		channel.Title = "Chat " + strconv.FormatInt(chat.ID, 10)
	}
	if c.channels == nil {
		c.channels = make(map[int64]*TelegramChannel)
	}
	c.channels[chat.ID] = channel
	c.Logger().Infof("Tracked channel: %s (ID: %d, Type: %s)", channel.Title, channel.ID, channel.Type)

	// Process message content
	messageText := message.Text
	if messageText == "" {
		messageText = message.Caption
	}

	c.Logger().Infof("Message text: %q", messageText)
	c.Logger().Infof("Message from: %+v", message.From)
	c.Logger().Infof("AMQP enabled: %v", runtimeconfig.Get().Amqp.Enabled)

	author := "unknown"
	if message.From != nil {
		if message.From.Username != "" {
			author = message.From.Username
		} else if message.From.FirstName != "" {
			author = message.From.FirstName
		}
	}

	incomingMsg := &msgs.IncomingMessage{
		Author:    author,
		Content:   messageText,
		Channel:   strconv.FormatInt(chat.ID, 10),
		Protocol:  "telegram",
		Timestamp: time.Now().Unix(),
		Identity:  c.nickname, // Include bot identity so hooks can route responses correctly
	}

	// Execute hooks if configured
	if len(c.hooks) > 0 {
		c.Logger().Debugf("Executing %d hook(s) for incoming message", len(c.hooks))
		hooks.ExecuteHooks(c.hooks, incomingMsg, c.Logger())
	}

	// Publish to AMQP if enabled
	if runtimeconfig.Get().Amqp.Enabled {
		c.Logger().Infof("Publishing message to AMQP - Author: %s, Content: %q, Channel: %d",
			author, messageText, chat.ID)

		amqp.PublishPb(incomingMsg)

		c.Logger().Infof("Message published to AMQP successfully")
	} else {
		c.Logger().Warnf("AMQP is disabled - message not published")
	}
}

func (c *TelegramConnector) Replier() {
	ctx := context.Background()

	// Use identity-specific routing key so this bot only receives messages intended for it
	// Note: nickname must be set before Replier() is called
	if c.nickname == "" {
		c.Logger().Errorf("Replier() called but nickname is not set! Cannot bind to identity-specific routing key.")
		return
	}

	routingKey := amqp.GetOutgoingMessageRoutingKey("telegram", c.nickname)
	c.Logger().Infof("Starting Replier for routing key: %s (bot identity: %s)", routingKey, c.nickname)

	// Consume from identity-specific routing key
	amqp.ConsumeForever(routingKey, func(d amqp.Delivery) {
		reply := msgs.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		c.Logger().Infof("Received OutgoingMessage - Protocol: %s, Identity: %s, Channel: %s, Content: %s",
			reply.Protocol, reply.Identity, reply.Channel, reply.Content)

		// Only process messages for this protocol
		// Note: AMQP binding should already filter by identity via routing key, but we keep this check for safety
		if reply.Protocol != "telegram" {
			c.Logger().Debugf("Ignoring message for protocol: %s", reply.Protocol)
			return
		}

		channelId, _ := strconv.ParseInt(reply.Channel, 10, 64)
		if c.botInstance != nil {
			c.botInstance.SendMessage(ctx, &tgbotapi.SendMessageParams{
				ChatID: channelId,
				Text:   reply.Content,
			})
		} else {
			c.Logger().Warnf("Bot instance is nil, cannot send message")
		}
	})
}

// GetChannels returns a list of all known channels/chats for this bot
// Implements ConnectorWithChannelsInfo interface
func (c *TelegramConnector) GetChannels() []*connector.BotChannel {
	if c.channels == nil {
		return []*connector.BotChannel{}
	}

	channels := make([]*connector.BotChannel, 0, len(c.channels))
	for _, channel := range c.channels {
		channels = append(channels, &connector.BotChannel{
			ID:       strconv.FormatInt(channel.ID, 10),
			Title:    channel.Title,
			Type:     channel.Type,
			Username: channel.Username,
		})
	}
	return channels
}

// GetHooks returns the configured hooks for this connector
// Implements ConnectorWithHooks interface
func (c *TelegramConnector) GetHooks() []*connector.IncomingMessageHook {
	hooks := make([]*connector.IncomingMessageHook, 0, len(c.hooks))
	for _, hook := range c.hooks {
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
func (c *TelegramConnector) SetHooks(hooks []*connector.IncomingMessageHook) error {
	c.hooks = make([]runtimeconfig.IncomingMessageHook, 0, len(hooks))
	for _, hook := range hooks {
		c.hooks = append(c.hooks, runtimeconfig.IncomingMessageHook{
			URL:     hook.URL,
			Enabled: hook.Enabled,
		})
	}
	c.Logger().Infof("Updated hooks: %d hook(s) configured", len(c.hooks))
	return nil
}

// RefreshChannelInfo attempts to get updated information about a channel using the Telegram API
func (c *TelegramConnector) RefreshChannelInfo(ctx context.Context, chatID int64) error {
	if c.botInstance == nil || !c.isRunning {
		return fmt.Errorf("bot is not running")
	}

	chat, err := c.botInstance.GetChat(ctx, &tgbotapi.GetChatParams{
		ChatID: chatID,
	})
	if err != nil {
		return fmt.Errorf("failed to get chat info: %w", err)
	}

	channel := &TelegramChannel{
		ID:       chat.ID,
		Title:    chat.Title,
		Type:     string(chat.Type),
		Username: chat.Username,
	}
	if channel.Title == "" && chat.FirstName != "" {
		channel.Title = chat.FirstName
		if chat.LastName != "" {
			channel.Title += " " + chat.LastName
		}
	}
	if channel.Title == "" && channel.Username != "" {
		channel.Title = "@" + channel.Username
	}
	if channel.Title == "" {
		channel.Title = "Chat " + strconv.FormatInt(chat.ID, 10)
	}

	if c.channels == nil {
		c.channels = make(map[int64]*TelegramChannel)
	}
	c.channels[chatID] = channel
	return nil
}
