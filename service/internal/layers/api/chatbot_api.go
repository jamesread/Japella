package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/chatbot"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/layers/authentication"
	log "github.com/sirupsen/logrus"
)

func marshalChatBot(bot *connector.ChatBot) *controlv1.ChatBot {
	if bot == nil {
		return nil
	}
	return &controlv1.ChatBot{
		BotId:               bot.BotID,
		Connector:           bot.Connector,
		Name:                bot.Name,
		Identity:            bot.Identity,
		Icon:                bot.Icon,
		IsRunning:           bot.IsRunning,
		StatusMessage:       bot.StatusMessage,
		ErrorMessage:        bot.ErrorMessage,
		ProtocolDisplayName: bot.ProtocolDisplayName,
	}
}

func (s *ControlApi) getChatBotControl(protocol, botID string) (connector.ConnectorWithChatBotControl, error) {
	if strings.TrimSpace(protocol) == "" || strings.TrimSpace(botID) == "" {
		return nil, fmt.Errorf("protocol and bot_id are required")
	}
	return s.cc.GetChatBotControl(protocol, botID)
}

func (s *ControlApi) ensureChatBotInstance(protocol, botID string) (*db.ChatBotInstance, error) {
	inst, err := s.DB.GetChatBotInstance(protocol, botID)
	if err != nil {
		return nil, fmt.Errorf("chat bot not found: %s/%s", protocol, botID)
	}
	return inst, nil
}

func (s *ControlApi) insertTelegramCvars(botID, token string) error {
	if token == "" {
		return fmt.Errorf("telegram bot token is required")
	}
	return s.DB.InsertBotPasswordCvar(
		chatbot.TelegramBotTokenKey(botID),
		fmt.Sprintf("Telegram bot token (%s)", botID),
		"Bot token from @BotFather",
		token,
	)
}

func (s *ControlApi) insertDiscordOptionalText(botID, appID, publicKey string) error {
	if appID != "" {
		if err := s.DB.InsertBotTextCvar(
			chatbot.DiscordAppIDKey(botID),
			fmt.Sprintf("Discord app ID (%s)", botID),
			"Discord application ID",
			appID,
		); err != nil {
			return err
		}
	}
	if publicKey == "" {
		return nil
	}
	return s.DB.InsertBotTextCvar(
		chatbot.DiscordPublicKeyKey(botID),
		fmt.Sprintf("Discord public key (%s)", botID),
		"Discord application public key",
		publicKey,
	)
}

func (s *ControlApi) insertDiscordCvars(botID, token, appID, publicKey string) error {
	if token == "" {
		return fmt.Errorf("discord bot token is required")
	}
	if err := s.DB.InsertBotPasswordCvar(
		chatbot.DiscordTokenKey(botID),
		fmt.Sprintf("Discord bot token (%s)", botID),
		"Discord bot token",
		token,
	); err != nil {
		return err
	}
	return s.insertDiscordOptionalText(botID, appID, publicKey)
}

func (s *ControlApi) insertChatBotCvars(protocol, botID string, telegramToken, discordToken, discordAppID, discordPublicKey string) error {
	switch protocol {
	case "telegram":
		return s.insertTelegramCvars(botID, telegramToken)
	case "discord":
		return s.insertDiscordCvars(botID, discordToken, discordAppID, discordPublicKey)
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func (s *ControlApi) updateTelegramCvars(botID, token string) error {
	if token == "" {
		return nil
	}
	return s.DB.InsertBotPasswordCvar(
		chatbot.TelegramBotTokenKey(botID),
		fmt.Sprintf("Telegram bot token (%s)", botID),
		"Bot token from @BotFather",
		token,
	)
}

func (s *ControlApi) updateDiscordCvars(botID, token, appID, publicKey string) error {
	if token != "" {
		if err := s.DB.InsertBotPasswordCvar(
			chatbot.DiscordTokenKey(botID),
			fmt.Sprintf("Discord bot token (%s)", botID),
			"Discord bot token",
			token,
		); err != nil {
			return err
		}
	}
	return s.insertDiscordOptionalText(botID, appID, publicKey)
}

func (s *ControlApi) updateChatBotCvars(protocol, botID string, telegramToken, discordToken, discordAppID, discordPublicKey string) error {
	switch protocol {
	case "telegram":
		return s.updateTelegramCvars(botID, telegramToken)
	case "discord":
		return s.updateDiscordCvars(botID, discordToken, discordAppID, discordPublicKey)
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func (s *ControlApi) reloadChatBotConnector(protocol, botID, displayName string) error {
	base := s.cc.Get(chatbot.ControllerKey(protocol, botID))
	if base == nil {
		return fmt.Errorf("chat bot not registered: %s/%s", protocol, botID)
	}
	cfg := &connector.ChatBotStartupConfig{
		Protocol:    protocol,
		BotID:       botID,
		DisplayName: displayName,
	}
	base.SetStartupConfiguration(&connector.ControllerStartupConfiguration{
		DB:     s.DB,
		Config: cfg,
	})
	return nil
}

func (s *ControlApi) GetChatBots(ctx context.Context, req *connect.Request[controlv1.GetChatBotsRequest]) (*connect.Response[controlv1.GetChatBotsResponse], error) {
	log.Infof("Fetching chatbots")

	bots := make([]*controlv1.ChatBot, 0)
	for _, svc := range s.cc.GetServices() {
		if chatbotConnector, ok := svc.(connector.ConnectorWithChatBot); ok {
			if bot := chatbotConnector.GetChatBot(); bot != nil && bot.BotID != "" {
				bots = append(bots, marshalChatBot(bot))
			}
		}
	}

	return connect.NewResponse(&controlv1.GetChatBotsResponse{Bots: bots}), nil
}

func validateNewChatBot(protocol, botID string) error {
	if err := chatbot.ValidateProtocol(protocol); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := chatbot.ValidateBotID(botID); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

func (s *ControlApi) persistNewChatBot(protocol, botID, displayName string) error {
	inst := &db.ChatBotInstance{Protocol: protocol, BotID: botID, DisplayName: displayName}
	if err := s.DB.CreateChatBotInstance(inst); err != nil {
		_ = s.DB.DeleteCvarsByKeys(chatbot.CvarKeysForInstance(protocol, botID))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create chat bot: %w", err))
	}
	if err := s.cc.RegisterChatBotInstance(protocol, botID, displayName); err != nil {
		_ = s.DB.DeleteChatBotInstance(protocol, botID)
		_ = s.DB.DeleteCvarsByKeys(chatbot.CvarKeysForInstance(protocol, botID))
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func chatBotDisplayNameOrID(displayName, botID string) string {
	if displayName == "" {
		return botID
	}
	return displayName
}

func (s *ControlApi) ensureChatBotAvailable(protocol, botID string) error {
	if _, err := s.DB.GetChatBotInstance(protocol, botID); err == nil {
		return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("chat bot already exists: %s/%s", protocol, botID))
	}
	return nil
}

func (s *ControlApi) CreateChatBot(ctx context.Context, req *connect.Request[controlv1.CreateChatBotRequest]) (*connect.Response[controlv1.CreateChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	displayName := chatBotDisplayNameOrID(strings.TrimSpace(req.Msg.DisplayName), botID)
	if err := validateNewChatBot(protocol, botID); err != nil {
		return nil, err
	}
	if err := s.ensureChatBotAvailable(protocol, botID); err != nil {
		return nil, err
	}
	if err := s.insertChatBotCvars(protocol, botID, req.Msg.TelegramBotToken, req.Msg.DiscordToken, req.Msg.DiscordAppId, req.Msg.DiscordPublicKey); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.persistNewChatBot(protocol, botID, displayName); err != nil {
		return nil, err
	}
	ctrl, _ := s.getChatBotControl(protocol, botID)
	return connect.NewResponse(&controlv1.CreateChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot created"},
		Bot:              marshalChatBot(ctrl.GetChatBot()),
	}), nil
}

func (s *ControlApi) applyChatBotDisplayName(inst *db.ChatBotInstance, protocol, botID, displayName string) error {
	if displayName == "" || displayName == inst.DisplayName {
		return nil
	}
	if err := s.DB.UpdateChatBotInstanceDisplayName(protocol, botID, displayName); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	inst.DisplayName = displayName
	return nil
}

func (s *ControlApi) UpdateChatBot(ctx context.Context, req *connect.Request[controlv1.UpdateChatBotRequest]) (*connect.Response[controlv1.UpdateChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	inst, err := s.ensureChatBotInstance(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := s.applyChatBotDisplayName(inst, protocol, botID, strings.TrimSpace(req.Msg.DisplayName)); err != nil {
		return nil, err
	}
	if err := s.updateChatBotCvars(protocol, botID, req.Msg.TelegramBotToken, req.Msg.DiscordToken, req.Msg.DiscordAppId, req.Msg.DiscordPublicKey); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.reloadChatBotConnector(protocol, botID, inst.DisplayName); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	ctrl, _ := s.getChatBotControl(protocol, botID)
	return connect.NewResponse(&controlv1.UpdateChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot updated"},
		Bot:              marshalChatBot(ctrl.GetChatBot()),
	}), nil
}

func (s *ControlApi) DeleteChatBot(ctx context.Context, req *connect.Request[controlv1.DeleteChatBotRequest]) (*connect.Response[controlv1.DeleteChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	if _, err := s.ensureChatBotInstance(protocol, botID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := s.cc.UnregisterChatBotInstance(protocol, botID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_ = s.DB.DeleteWebhookHooks(protocol, botID)
	_ = s.DB.DeleteOrphanedWebhookHooks()
	_ = s.DB.DeleteCvarsByKeys(chatbot.CvarKeysForInstance(protocol, botID))
	if err := s.DB.DeleteChatBotInstance(protocol, botID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlv1.DeleteChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot deleted"},
	}), nil
}

func (s *ControlApi) StartChatBot(ctx context.Context, req *connect.Request[controlv1.StartChatBotRequest]) (*connect.Response[controlv1.StartChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := ctrl.StartChatBot(); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&controlv1.StartChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot started"},
		Bot:              marshalChatBot(ctrl.GetChatBot()),
	}), nil
}

func (s *ControlApi) StopChatBot(ctx context.Context, req *connect.Request[controlv1.StopChatBotRequest]) (*connect.Response[controlv1.StopChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := ctrl.StopChatBot(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.StopChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot stopped"},
		Bot:              marshalChatBot(ctrl.GetChatBot()),
	}), nil
}

func (s *ControlApi) GetBotChannels(ctx context.Context, req *connect.Request[controlv1.GetBotChannelsRequest]) (*connect.Response[controlv1.GetBotChannelsResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	channels := make([]*controlv1.BotChannel, 0)
	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if channelsConnector, ok := ctrl.(connector.ConnectorWithChannelsInfo); ok {
		for _, ch := range channelsConnector.GetChannels() {
			channels = append(channels, &controlv1.BotChannel{
				Id: ch.ID, Title: ch.Title, Type: ch.Type, Username: ch.Username,
			})
		}
	}
	return connect.NewResponse(&controlv1.GetBotChannelsResponse{Channels: channels}), nil
}

func (s *ControlApi) GetBotHooks(ctx context.Context, req *connect.Request[controlv1.GetBotHooksRequest]) (*connect.Response[controlv1.GetBotHooksResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	if _, err := s.ensureChatBotInstance(protocol, botID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	hooks := make([]*controlv1.IncomingMessageHook, 0)
	dbHooks, err := s.DB.SelectWebhookHooks(protocol, botID)
	if err == nil {
		for _, hook := range dbHooks {
			hooks = append(hooks, &controlv1.IncomingMessageHook{Url: hook.URL, Enabled: hook.Enabled, SkipTlsVerify: hook.SkipTLSVerify})
		}
	}
	return connect.NewResponse(&controlv1.GetBotHooksResponse{Hooks: hooks}), nil
}

func chatBotIdentity(ctrl connector.ConnectorWithChatBotControl) string {
	if bot := ctrl.GetChatBot(); bot != nil {
		return bot.Identity
	}
	return ""
}

func (s *ControlApi) replaceWebhookHooks(protocol, botID, platformIdentity string, hooks []*controlv1.IncomingMessageHook) error {
	if err := s.DB.DeleteWebhookHooks(protocol, botID); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	for _, hook := range hooks {
		if err := s.DB.CreateWebhookHook(&db.WebhookHook{
			Connector:     protocol,
			Identity:      platformIdentity,
			BotID:         botID,
			URL:           hook.Url,
			Enabled:       hook.Enabled,
			SkipTLSVerify: hook.SkipTlsVerify,
		}); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	return nil
}

func applyHooksToConnector(ctrl connector.ConnectorWithChatBotControl, hooks []*controlv1.IncomingMessageHook) {
	hooksConnector, ok := ctrl.(connector.ConnectorWithHooks)
	if !ok {
		return
	}
	connectorHooks := make([]*connector.IncomingMessageHook, 0, len(hooks))
	for _, hook := range hooks {
		connectorHooks = append(connectorHooks, &connector.IncomingMessageHook{
			URL: hook.Url, Enabled: hook.Enabled, SkipTLSVerify: hook.SkipTlsVerify,
		})
	}
	_ = hooksConnector.SetHooks(connectorHooks)
}

func (s *ControlApi) SetBotHooks(ctx context.Context, req *connect.Request[controlv1.SetBotHooksRequest]) (*connect.Response[controlv1.SetBotHooksResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := s.replaceWebhookHooks(protocol, botID, chatBotIdentity(ctrl), req.Msg.Hooks); err != nil {
		return nil, err
	}
	applyHooksToConnector(ctrl, req.Msg.Hooks)

	return connect.NewResponse(&controlv1.SetBotHooksResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Hooks updated successfully"},
	}), nil
}

func botConversationLimit(requested, fallback int) int {
	if requested <= 0 {
		return fallback
	}
	return requested
}

func marshalBotConversations(rows []*db.ChatBotConversationSummary) []*controlv1.BotConversation {
	conversations := make([]*controlv1.BotConversation, 0, len(rows))
	for _, row := range rows {
		conversations = append(conversations, &controlv1.BotConversation{
			Key: row.ConversationKey, Title: row.ConversationTitle,
			LastMessage: row.LastMessage, LastDirection: row.LastDirection,
			LastMessageAtUnix: row.LastMessageAtUnix,
		})
	}
	return conversations
}

func (s *ControlApi) GetBotConversations(ctx context.Context, req *connect.Request[controlv1.GetBotConversationsRequest]) (*connect.Response[controlv1.GetBotConversationsResponse], error) {
	if err := requireAdminUser(s.getAuthenticatedUser(ctx)); err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	rows, err := s.DB.SelectChatBotConversations(protocol, botID, botConversationLimit(int(req.Msg.Limit), 100))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.GetBotConversationsResponse{Conversations: marshalBotConversations(rows)}), nil
}

func (s *ControlApi) GetBotConversationMessages(ctx context.Context, req *connect.Request[controlv1.GetBotConversationMessagesRequest]) (*connect.Response[controlv1.GetBotConversationMessagesResponse], error) {
	if err := requireAdminUser(s.getAuthenticatedUser(ctx)); err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	rows, err := s.DB.SelectChatBotConversationMessages(protocol, botID, req.Msg.ConversationKey, botConversationLimit(int(req.Msg.Limit), 200))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.GetBotConversationMessagesResponse{
		Messages: marshalBotConversationMessageRows(rows),
	}), nil
}

func marshalBotConversationMessageRows(rows []*db.ChatBotMessage) []*controlv1.BotConversationMessage {
	messages := make([]*controlv1.BotConversationMessage, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, &controlv1.BotConversationMessage{
			Id: row.ID, Author: row.Author, Content: row.Content,
			Direction: row.Direction, Channel: row.Channel,
			MessageId: row.MessageID, TimestampUnix: row.TimestampUnix,
		})
	}
	return messages
}

func requireAdminUser(au *authentication.AuthenticatedUser) error {
	if au == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if !au.CanAccessControlPanel() {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}
	return nil
}

func (s *ControlApi) streamConversationTick(protocol, botID, conversationKey string, cursor *uint32, stream *connect.ServerStream[controlv1.StreamBotConversationUpdatesResponse]) error {
	rows, err := s.DB.SelectChatBotConversationMessagesAfterID(protocol, botID, conversationKey, *cursor, 100)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if len(rows) == 0 {
		return nil
	}
	if err := stream.Send(&controlv1.StreamBotConversationUpdatesResponse{
		NewMessages: marshalBotConversationMessageRows(rows),
	}); err != nil {
		return err
	}
	*cursor = rows[len(rows)-1].ID
	return nil
}

func validateStreamConversationRequest(protocol, botID, conversationKey string) error {
	if protocol == "" || botID == "" || strings.TrimSpace(conversationKey) == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("protocol, bot_id and conversation_key are required"))
	}
	return nil
}

func (s *ControlApi) runConversationStream(ctx context.Context, protocol, botID, conversationKey string, cursor uint32, stream *connect.ServerStream[controlv1.StreamBotConversationUpdatesResponse]) error {
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.streamConversationTick(protocol, botID, conversationKey, &cursor, stream); err != nil {
				return err
			}
		}
	}
}

func (s *ControlApi) StreamBotConversationUpdates(ctx context.Context, req *connect.Request[controlv1.StreamBotConversationUpdatesRequest], stream *connect.ServerStream[controlv1.StreamBotConversationUpdatesResponse]) error {
	if err := requireAdminUser(s.getAuthenticatedUser(ctx)); err != nil {
		return err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	if err := validateStreamConversationRequest(protocol, botID, req.Msg.ConversationKey); err != nil {
		return err
	}
	return s.runConversationStream(ctx, protocol, botID, req.Msg.ConversationKey, req.Msg.LastMessageId, stream)
}

func conversationChannel(conversationKey string) (string, error) {
	parts := strings.SplitN(conversationKey, "|", 2)
	channel := ""
	if len(parts) > 0 {
		channel = parts[0]
	}
	if channel == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid conversation key"))
	}
	return channel, nil
}

func botOfflineMessage(bot *connector.ChatBot) string {
	if strings.TrimSpace(bot.StatusMessage) != "" {
		return fmt.Sprintf("Bot is offline: %s", bot.StatusMessage)
	}
	return "Bot is offline; cannot send replies"
}

func (s *ControlApi) sendBotMessageIdentity(ctrl connector.ConnectorWithChatBotControl) (string, *connect.Response[controlv1.SendBotConversationMessageResponse]) {
	bot := ctrl.GetChatBot()
	if bot == nil {
		return "", nil
	}
	if bot.IsRunning {
		return bot.Identity, nil
	}
	return "", connect.NewResponse(&controlv1.SendBotConversationMessageResponse{
		StandardResponse: &controlv1.StandardResponse{Success: false, Message: botOfflineMessage(bot)},
	})
}

func requireNonEmptyContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is required"))
	}
	return nil
}

func (s *ControlApi) publishBotConversationMessage(protocol, botID, conversationKey, content string) (*connect.Response[controlv1.SendBotConversationMessageResponse], error) {
	channel, err := conversationChannel(conversationKey)
	if err != nil {
		return nil, err
	}
	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	platformIdentity, offline := s.sendBotMessageIdentity(ctrl)
	if offline != nil {
		return offline, nil
	}
	outgoing := &msgs.OutgoingMessage{
		Content:         content,
		Channel:         channel,
		Protocol:        protocol,
		Identity:        platformIdentity,
		ConversationKey: strings.TrimSpace(conversationKey),
	}
	amqp.PublishPbWithRoutingKey(outgoing, amqp.GetOutgoingMessageRoutingKey(protocol, platformIdentity))
	return connect.NewResponse(&controlv1.SendBotConversationMessageResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Message sent"},
	}), nil
}

func (s *ControlApi) SendBotConversationMessage(ctx context.Context, req *connect.Request[controlv1.SendBotConversationMessageRequest]) (*connect.Response[controlv1.SendBotConversationMessageResponse], error) {
	if err := requireAdminUser(s.getAuthenticatedUser(ctx)); err != nil {
		return nil, err
	}
	if err := requireNonEmptyContent(req.Msg.Content); err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	return s.publishBotConversationMessage(protocol, botID, req.Msg.ConversationKey, req.Msg.Content)
}
