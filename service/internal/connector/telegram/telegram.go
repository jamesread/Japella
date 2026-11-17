package telegram

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram/bot"
	tgbotmdl "github.com/go-telegram/bot/models"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/chatbot"
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
	botID               string
	nickname            string
	displayName         string
	protocolDisplayName string // FirstName from GetMe response
	isRunning           bool
	statusMessage       string
	errorMessage        string
	botToken            string
	runCancel           context.CancelFunc
	botMu               sync.Mutex
	channels            map[int64]*TelegramChannel          // Track known channels/chats
	botInstance         *tgbotapi.Bot                       // Store bot instance for API calls
	hooks               []runtimeconfig.IncomingMessageHook // Webhooks to call when messages are received
	db                  *db.DB                              // Database reference for loading hooks
	utils.LogComponent

	connector.BaseConnector
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
	return &connector.ChatBot{
		BotID:               c.botID,
		Connector:           c.GetProtocol(),
		Name:                c.chatBotName(),
		Identity:            c.nickname,
		Icon:                c.GetIcon(),
		IsRunning:           c.isRunning,
		StatusMessage:       chatBotStatusMessage(c.isRunning, c.errorMessage, c.statusMessage),
		ErrorMessage:        c.errorMessage,
		ProtocolDisplayName: c.protocolDisplayName,
	}
}

func (c *TelegramConnector) chatBotName() string {
	name := c.GetDisplayName()
	if name == c.GetProtocol() && c.nickname != "" {
		return c.nickname
	}
	return name
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

func (c *TelegramConnector) GetBotID() string {
	return c.botID
}

func (c *TelegramConnector) failStartup(statusMsg, errMsg string) {
	c.isRunning = false
	c.statusMessage = statusMsg
	c.errorMessage = errMsg
	connector.LogChatBotStartupFailure(c.db, c.GetProtocol(), c.nickname, errMsg)
}

func (c *TelegramConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	c.db = startup.DB
	cfg, _ := startup.Config.(*connector.ChatBotStartupConfig)
	if cfg == nil || cfg.BotID == "" {
		c.failStartup("Configuration error: Bot instance is not configured", "Chat bot instance configuration is missing")
		return
	}

	c.botID = cfg.BotID
	c.displayName = cfg.DisplayName
	c.botToken = c.db.GetCvarString(chatbot.TelegramBotTokenKey(cfg.BotID))
	if c.botToken == "" {
		c.failStartup("Configuration error: Bot token is not set", "Telegram bot token is missing; set it when creating or editing the bot")
		return
	}

	c.isRunning = false
	c.statusMessage = "Stopped (not connected)"
	c.errorMessage = ""
}

func (c *TelegramConnector) Start() {
	// Bot connection is started explicitly via StartChatBot
}

func (c *TelegramConnector) StartChatBot() error {
	c.botMu.Lock()
	defer c.botMu.Unlock()

	if c.isRunning {
		return fmt.Errorf("telegram bot is already running")
	}
	if err := c.ensureBotToken(); err != nil {
		return err
	}

	c.SetPrefix("Telegram-new")
	c.Logger().Infof("japella-bot-telegram")
	c.startBot(c.botToken)
	if !c.isRunning {
		return c.startFailureError("failed to start telegram bot")
	}
	return nil
}

func (c *TelegramConnector) ensureBotToken() error {
	if c.botToken == "" {
		c.botToken = c.db.GetCvarString(chatbot.TelegramBotTokenKey(c.botID))
	}
	if c.botToken == "" {
		return fmt.Errorf("telegram bot token is not configured")
	}
	return nil
}

func (c *TelegramConnector) startFailureError(fallback string) error {
	if c.errorMessage != "" {
		return fmt.Errorf("%s", c.errorMessage)
	}
	return fmt.Errorf("%s", fallback)
}

func (c *TelegramConnector) StopChatBot() error {
	c.botMu.Lock()
	defer c.botMu.Unlock()

	if !c.isRunning {
		c.statusMessage = "Stopped (not connected)"
		c.errorMessage = ""
		return nil
	}

	if c.runCancel != nil {
		c.runCancel()
		c.runCancel = nil
	}

	c.botInstance = nil
	c.isRunning = false
	c.statusMessage = "Stopped (not connected)"
	c.errorMessage = ""
	c.Logger().Infof("Telegram bot stopped")
	return nil
}

func (c *TelegramConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	// Telegram uses bot tokens, not OAuth, so no refresh is needed
	return nil
}

func (c *TelegramConnector) startBot(botToken string) {
	c.Logger().Infof("Starting Telegram bot with token (length: %d)", len(botToken))
	c.ensureChannelsMap()

	opts := []tgbotapi.Option{
		tgbotapi.WithDefaultHandler(c.debugHandlerWrapper),
	}
	c.Logger().Infof("Creating bot instance with %d options", len(opts))

	var err error
	c.botInstance, err = tgbotapi.New(botToken, opts...)
	if err != nil {
		c.Logger().Errorf("Error creating bot: %v", err)
		c.failStartup("Failed to initialize bot connection", "Failed to create bot instance: "+err.Error())
		return
	}
	c.Logger().Infof("Bot instance created successfully")

	me, err := c.verifyBotIdentity()
	if err != nil {
		return
	}
	c.markBotRunning(me)
	c.startAMQPReplierIfEnabled()
	c.loadHooksFromDB()
	c.sendDebugStartupIfConfigured(me.Username)
	c.startPollingLoop()
	c.Logger().Infof("Bot startup complete - isRunning: %v", c.isRunning)
}

func (c *TelegramConnector) ensureChannelsMap() {
	if c.channels == nil {
		c.channels = make(map[int64]*TelegramChannel)
	}
}

func (c *TelegramConnector) debugHandlerWrapper(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
	c.Logger().Infof("=== MESSAGE HANDLER CALLED ===")
	c.Logger().Infof("Update type: Message=%v, CallbackQuery=%v, ChannelPost=%v",
		update.Message != nil,
		update.CallbackQuery != nil,
		update.ChannelPost != nil)
	if update.Message != nil {
		c.Logger().Infof("Message details - ChatID: %d, From: %+v, Text: %s",
			update.Message.Chat.ID,
			update.Message.From,
			update.Message.Text)
	}
	c.messageHandler(update)
}

func (c *TelegramConnector) verifyBotIdentity() (*tgbotmdl.User, error) {
	getMeCtx, getMeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer getMeCancel()

	c.Logger().Infof("Calling GetMe to verify bot token")
	me, err := c.botInstance.GetMe(getMeCtx)
	if err != nil {
		c.Logger().Errorf("Error getting bot info: %v", err)
		c.failStartup("Bot token validation failed", "Failed to verify bot token: "+err.Error())
		return nil, err
	}
	return me, nil
}

func (c *TelegramConnector) markBotRunning(me *tgbotmdl.User) {
	c.nickname = me.Username
	c.protocolDisplayName = me.FirstName
	c.isRunning = true
	c.statusMessage = "Bot is running and connected"
	c.errorMessage = ""
	c.SetPrefix("Telegram-" + c.nickname)
	c.Logger().Infof("Bot verified successfully - Username: %s, ID: %d, FirstName: %s, CanJoinGroups: %v, CanReadAllGroupMessages: %v",
		me.Username, me.ID, me.FirstName, me.CanJoinGroups, me.CanReadAllGroupMessages)
}

func (c *TelegramConnector) startAMQPReplierIfEnabled() {
	if !runtimeconfig.Get().Amqp.Enabled {
		return
	}
	c.Logger().Infof("Starting AMQP replier with identity: %s", c.nickname)
	go c.Replier()
}

func (c *TelegramConnector) loadHooksFromDB() {
	if c.db == nil {
		return
	}
	dbHooks, err := c.db.SelectWebhookHooks("telegram", c.botID)
	if err != nil {
		c.Logger().Warnf("Failed to load hooks from database: %v, using config hooks", err)
		return
	}
	if len(dbHooks) > 0 {
		c.hooks = hooks.FromDB(dbHooks)
		c.Logger().Infof("Loaded %d incoming message hook(s) from database", len(c.hooks))
	}
}

func (c *TelegramConnector) sendDebugStartupIfConfigured(botUsername string) {
	debugChatId := runtimeconfig.Get().TelegramDebugChatId
	if debugChatId == "" {
		return
	}
	debugCtx, debugCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer debugCancel()
	c.sendDebugStartupMessage(debugCtx, debugChatId, botUsername)
}

func (c *TelegramConnector) startPollingLoop() {
	startCtx, startCancel := context.WithCancel(context.Background())
	c.runCancel = startCancel

	c.Logger().Infof("Starting bot polling in goroutine...")
	go func() {
		defer startCancel()
		c.Logger().Infof("Bot.Start() called - beginning to poll for updates")
		c.botInstance.Start(startCtx)
		c.Logger().Infof("Bot.Start() returned (context cancelled)")

		c.botMu.Lock()
		defer c.botMu.Unlock()
		c.isRunning = false
		c.statusMessage = "Stopped (not connected)"
		c.botInstance = nil
		c.runCancel = nil
	}()
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

func (c *TelegramConnector) messageHandler(update *tgbotmdl.Update) {
	c.Logger().Infof("=== messageHandler ENTERED ===")
	c.dispatchUpdate(update)
	c.Logger().Infof("=== messageHandler EXITING ===")
}

func (c *TelegramConnector) dispatchUpdate(update *tgbotmdl.Update) {
	if msg := firstMessageInUpdate(update); msg != nil {
		c.Logger().Infof("Processing message-like update")
		c.handleMessage(msg)
		return
	}
	c.logNonMessageUpdate(update)
}

func firstMessageInUpdate(update *tgbotmdl.Update) *tgbotmdl.Message {
	switch {
	case update.Message != nil:
		return update.Message
	case update.ChannelPost != nil:
		return update.ChannelPost
	case update.EditedMessage != nil:
		return update.EditedMessage
	default:
		return nil
	}
}

func (c *TelegramConnector) logNonMessageUpdate(update *tgbotmdl.Update) {
	switch {
	case update.CallbackQuery != nil:
		c.Logger().Infof("Processing CallbackQuery update (ID: %s)", update.CallbackQuery.ID)
	case update.InlineQuery != nil:
		c.Logger().Infof("Processing InlineQuery update (ID: %s)", update.InlineQuery.ID)
	default:
		c.Logger().Warnf("Received update with unknown type")
		c.Logger().Infof("Full update structure: %+v", update)
	}
}

func (c *TelegramConnector) handleMessage(message *tgbotmdl.Message) {
	c.Logger().Infof("handleMessage called - ChatID: %d", message.Chat.ID)
	channel := c.trackChannelFromChat(message.Chat)
	messageText := message.Text
	if messageText == "" {
		messageText = message.Caption
	}
	c.Logger().Infof("Message text: %q", messageText)
	c.Logger().Infof("Message from: %+v", message.From)
	c.Logger().Infof("AMQP enabled: %v", runtimeconfig.Get().Amqp.Enabled)

	author := telegramMessageAuthor(message)
	incomingMsg := &msgs.IncomingMessage{
		Author:    author,
		Content:   messageText,
		Channel:   strconv.FormatInt(message.Chat.ID, 10),
		Protocol:  "telegram",
		MessageId: telegramMessageID(message.ID),
		Timestamp: time.Now().Unix(),
		Identity:  c.nickname,
	}
	c.logIncomingTelegramMessage(incomingMsg, channel.Title, author)
	c.executeTelegramHooks(incomingMsg)
	c.publishIncomingIfEnabled(incomingMsg, author, messageText, message.Chat.ID)
}

func (c *TelegramConnector) trackChannelFromChat(chat tgbotmdl.Chat) *TelegramChannel {
	channel := &TelegramChannel{
		ID:       chat.ID,
		Title:    resolveChatTitle(chat.ID, chat.Title, chat.FirstName, chat.LastName, chat.Username),
		Type:     string(chat.Type),
		Username: chat.Username,
	}
	c.ensureChannelsMap()
	c.channels[chat.ID] = channel
	c.Logger().Infof("Tracked channel: %s (ID: %d, Type: %s)", channel.Title, channel.ID, channel.Type)
	return channel
}

func resolveChatTitle(id int64, title, firstName, lastName, username string) string {
	if title != "" {
		return title
	}
	if firstName != "" {
		return joinPersonName(firstName, lastName)
	}
	if username != "" {
		return "@" + username
	}
	return "Chat " + strconv.FormatInt(id, 10)
}

func joinPersonName(firstName, lastName string) string {
	if lastName == "" {
		return firstName
	}
	return firstName + " " + lastName
}

func telegramMessageAuthor(message *tgbotmdl.Message) string {
	if message.From == nil {
		return "unknown"
	}
	if message.From.Username != "" {
		return message.From.Username
	}
	if message.From.FirstName != "" {
		return message.From.FirstName
	}
	return "unknown"
}

func telegramMessageID(id int) string {
	if id == 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func (c *TelegramConnector) logIncomingTelegramMessage(msg *msgs.IncomingMessage, channelTitle, author string) {
	if c.db == nil {
		return
	}
	convTitle := channelTitle
	if convTitle == "" {
		convTitle = "Chat " + msg.Channel
	}
	_ = c.db.InsertChatBotMessage(&db.ChatBotMessage{
		Connector:         "telegram",
		Identity:          c.nickname,
		BotID:             c.botID,
		ConversationKey:   db.BuildConversationKey(msg.Channel, ""),
		ConversationTitle: convTitle,
		Channel:           msg.Channel,
		Author:            author,
		Content:           msg.Content,
		Direction:         "incoming",
		MessageID:         msg.MessageId,
		TimestampUnix:     msg.Timestamp,
	})
}

func (c *TelegramConnector) executeTelegramHooks(msg *msgs.IncomingMessage) {
	if len(c.hooks) == 0 {
		return
	}
	c.Logger().Debugf("Executing %d hook(s) for incoming message", len(c.hooks))
	hooks.ExecuteHooks(c.hooks, msg, c.Logger())
}

func (c *TelegramConnector) publishIncomingIfEnabled(msg *msgs.IncomingMessage, author, messageText string, chatID int64) {
	if !runtimeconfig.Get().Amqp.Enabled {
		c.Logger().Warnf("AMQP is disabled - message not published")
		return
	}
	c.Logger().Infof("Publishing message to AMQP - Author: %s, Content: %q, Channel: %d",
		author, messageText, chatID)
	amqp.PublishPb(msg)
	c.Logger().Infof("Message published to AMQP successfully")
}

func (c *TelegramConnector) Replier() {
	ctx := context.Background()
	if c.nickname == "" {
		c.Logger().Errorf("Replier() called but nickname is not set! Cannot bind to identity-specific routing key.")
		return
	}

	routingKey := amqp.GetOutgoingMessageRoutingKey("telegram", c.nickname)
	c.Logger().Infof("Starting Replier for routing key: %s (bot identity: %s)", routingKey, c.nickname)

	amqp.ConsumeForever(routingKey, func(d amqp.Delivery) {
		c.handleOutgoingDelivery(ctx, d)
	})
}

func (c *TelegramConnector) handleOutgoingDelivery(ctx context.Context, d amqp.Delivery) {
	reply := msgs.OutgoingMessage{}
	if err := amqp.Decode(d.Message.Body, &reply); err != nil {
		c.Logger().Errorf("decode OutgoingMessage: %v", err)
		return
	}
	c.Logger().Infof("Received OutgoingMessage - Protocol: %s, Identity: %s, Channel: %s, Content: %s",
		reply.Protocol, reply.Identity, reply.Channel, reply.Content)
	if reply.Protocol != "telegram" {
		c.Logger().Debugf("Ignoring message for protocol: %s", reply.Protocol)
		return
	}
	c.sendOutgoingReply(ctx, &reply)
}

func (c *TelegramConnector) sendOutgoingReply(ctx context.Context, reply *msgs.OutgoingMessage) {
	channelID, err := strconv.ParseInt(reply.Channel, 10, 64)
	if err != nil {
		c.Logger().Errorf("OutgoingMessage has invalid channel %q: %v", reply.Channel, err)
		return
	}
	if c.botInstance == nil {
		c.Logger().Warnf("Bot instance is nil, cannot send message")
		return
	}
	_, sendErr := c.botInstance.SendMessage(ctx, &tgbotapi.SendMessageParams{
		ChatID: channelID,
		Text:   reply.Content,
	})
	if sendErr != nil {
		c.Logger().Errorf("SendMessage failed (chat_id=%d): %v", channelID, sendErr)
		return
	}
	c.logOutgoingTelegramMessage(reply)
}

func (c *TelegramConnector) logOutgoingTelegramMessage(reply *msgs.OutgoingMessage) {
	if c.db == nil {
		return
	}
	conversationKey, conversationTitle := c.db.ResolveOutgoingConversationForLog(
		"telegram", c.botID, reply.Channel,
		reply.GetConversationKey(), reply.GetIncommingMessageId(),
	)
	_ = c.db.InsertChatBotMessage(&db.ChatBotMessage{
		Connector:         "telegram",
		Identity:          c.nickname,
		BotID:             c.botID,
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
func (c *TelegramConnector) SetHooks(hooks []*connector.IncomingMessageHook) error {
	c.hooks = make([]runtimeconfig.IncomingMessageHook, 0, len(hooks))
	for _, hook := range hooks {
		c.hooks = append(c.hooks, runtimeconfig.IncomingMessageHook{
			URL:           hook.URL,
			Enabled:       hook.Enabled,
			SkipTLSVerify: hook.SkipTLSVerify,
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

	c.ensureChannelsMap()
	c.channels[chatID] = &TelegramChannel{
		ID:       chat.ID,
		Title:    resolveChatTitle(chat.ID, chat.Title, chat.FirstName, chat.LastName, chat.Username),
		Type:     string(chat.Type),
		Username: chat.Username,
	}
	return nil
}
