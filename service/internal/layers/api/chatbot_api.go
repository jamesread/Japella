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

func (s *ControlApi) insertChatBotCvars(protocol, botID string, telegramToken, discordToken, discordAppID, discordPublicKey string) error {
	switch protocol {
	case "telegram":
		if telegramToken == "" {
			return fmt.Errorf("telegram bot token is required")
		}
		return s.DB.InsertBotPasswordCvar(
			chatbot.TelegramBotTokenKey(botID),
			fmt.Sprintf("Telegram bot token (%s)", botID),
			"Bot token from @BotFather",
			telegramToken,
		)
	case "discord":
		if discordToken == "" {
			return fmt.Errorf("discord bot token is required")
		}
		if err := s.DB.InsertBotPasswordCvar(
			chatbot.DiscordTokenKey(botID),
			fmt.Sprintf("Discord bot token (%s)", botID),
			"Discord bot token",
			discordToken,
		); err != nil {
			return err
		}
		if discordAppID != "" {
			if err := s.DB.InsertBotTextCvar(
				chatbot.DiscordAppIDKey(botID),
				fmt.Sprintf("Discord app ID (%s)", botID),
				"Discord application ID",
				discordAppID,
			); err != nil {
				return err
			}
		}
		if discordPublicKey != "" {
			if err := s.DB.InsertBotTextCvar(
				chatbot.DiscordPublicKeyKey(botID),
				fmt.Sprintf("Discord public key (%s)", botID),
				"Discord application public key",
				discordPublicKey,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func (s *ControlApi) updateChatBotCvars(protocol, botID string, telegramToken, discordToken, discordAppID, discordPublicKey string) error {
	switch protocol {
	case "telegram":
		if telegramToken != "" {
			return s.DB.InsertBotPasswordCvar(
				chatbot.TelegramBotTokenKey(botID),
				fmt.Sprintf("Telegram bot token (%s)", botID),
				"Bot token from @BotFather",
				telegramToken,
			)
		}
	case "discord":
		if discordToken != "" {
			if err := s.DB.InsertBotPasswordCvar(
				chatbot.DiscordTokenKey(botID),
				fmt.Sprintf("Discord bot token (%s)", botID),
				"Discord bot token",
				discordToken,
			); err != nil {
				return err
			}
		}
		if discordAppID != "" {
			if err := s.DB.InsertBotTextCvar(
				chatbot.DiscordAppIDKey(botID),
				fmt.Sprintf("Discord app ID (%s)", botID),
				"Discord application ID",
				discordAppID,
			); err != nil {
				return err
			}
		}
		if discordPublicKey != "" {
			if err := s.DB.InsertBotTextCvar(
				chatbot.DiscordPublicKeyKey(botID),
				fmt.Sprintf("Discord public key (%s)", botID),
				"Discord application public key",
				discordPublicKey,
			); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
	return nil
}

func (s *ControlApi) reloadChatBotConnector(protocol, botID, displayName string) error {
	svc := s.cc.Get(chatbot.ControllerKey(protocol, botID))
	if svc == nil {
		return fmt.Errorf("chat bot not registered: %s/%s", protocol, botID)
	}
	base, ok := svc.(connector.BaseConnector)
	if !ok {
		return fmt.Errorf("connector does not support reload: %s/%s", protocol, botID)
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

func (s *ControlApi) CreateChatBot(ctx context.Context, req *connect.Request[controlv1.CreateChatBotRequest]) (*connect.Response[controlv1.CreateChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	displayName := strings.TrimSpace(req.Msg.DisplayName)

	if err := chatbot.ValidateProtocol(protocol); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := chatbot.ValidateBotID(botID); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if displayName == "" {
		displayName = botID
	}

	if _, err := s.DB.GetChatBotInstance(protocol, botID); err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("chat bot already exists: %s/%s", protocol, botID))
	}

	if err := s.insertChatBotCvars(protocol, botID, req.Msg.TelegramBotToken, req.Msg.DiscordToken, req.Msg.DiscordAppId, req.Msg.DiscordPublicKey); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	inst := &db.ChatBotInstance{
		Protocol:    protocol,
		BotID:       botID,
		DisplayName: displayName,
	}
	if err := s.DB.CreateChatBotInstance(inst); err != nil {
		_ = s.DB.DeleteCvarsByKeys(chatbot.CvarKeysForInstance(protocol, botID))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create chat bot: %w", err))
	}

	if err := s.cc.RegisterChatBotInstance(protocol, botID, displayName); err != nil {
		_ = s.DB.DeleteChatBotInstance(protocol, botID)
		_ = s.DB.DeleteCvarsByKeys(chatbot.CvarKeysForInstance(protocol, botID))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	ctrl, _ := s.getChatBotControl(protocol, botID)
	res := connect.NewResponse(&controlv1.CreateChatBotResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Chat bot created"},
		Bot:            marshalChatBot(ctrl.GetChatBot()),
	})
	return res, nil
}

func (s *ControlApi) UpdateChatBot(ctx context.Context, req *connect.Request[controlv1.UpdateChatBotRequest]) (*connect.Response[controlv1.UpdateChatBotResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	inst, err := s.ensureChatBotInstance(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	displayName := strings.TrimSpace(req.Msg.DisplayName)
	if displayName != "" && displayName != inst.DisplayName {
		if err := s.DB.UpdateChatBotInstanceDisplayName(protocol, botID, displayName); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		inst.DisplayName = displayName
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
		Bot:            marshalChatBot(ctrl.GetChatBot()),
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
			hooks = append(hooks, &controlv1.IncomingMessageHook{Url: hook.URL, Enabled: hook.Enabled})
		}
	}
	return connect.NewResponse(&controlv1.GetBotHooksResponse{Hooks: hooks}), nil
}

func (s *ControlApi) SetBotHooks(ctx context.Context, req *connect.Request[controlv1.SetBotHooksRequest]) (*connect.Response[controlv1.SetBotHooksResponse], error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	platformIdentity := ""
	if bot := ctrl.GetChatBot(); bot != nil {
		platformIdentity = bot.Identity
	}

	if err := s.DB.DeleteWebhookHooks(protocol, botID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, hook := range req.Msg.Hooks {
		if err := s.DB.CreateWebhookHook(&db.WebhookHook{
			Connector: protocol,
			Identity:  platformIdentity,
			BotID:     botID,
			URL:       hook.Url,
			Enabled:   hook.Enabled,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if hooksConnector, ok := ctrl.(connector.ConnectorWithHooks); ok {
		connectorHooks := make([]*connector.IncomingMessageHook, 0, len(req.Msg.Hooks))
		for _, hook := range req.Msg.Hooks {
			connectorHooks = append(connectorHooks, &connector.IncomingMessageHook{
				URL: hook.Url, Enabled: hook.Enabled,
			})
		}
		_ = hooksConnector.SetHooks(connectorHooks)
	}

	return connect.NewResponse(&controlv1.SetBotHooksResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Hooks updated successfully"},
	}), nil
}

func (s *ControlApi) GetBotConversations(ctx context.Context, req *connect.Request[controlv1.GetBotConversationsRequest]) (*connect.Response[controlv1.GetBotConversationsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if !au.CanAccessControlPanel() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.DB.SelectChatBotConversations(protocol, botID, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	conversations := make([]*controlv1.BotConversation, 0, len(rows))
	for _, row := range rows {
		conversations = append(conversations, &controlv1.BotConversation{
			Key: row.ConversationKey, Title: row.ConversationTitle,
			LastMessage: row.LastMessage, LastDirection: row.LastDirection,
			LastMessageAtUnix: row.LastMessageAtUnix,
		})
	}
	return connect.NewResponse(&controlv1.GetBotConversationsResponse{Conversations: conversations}), nil
}

func (s *ControlApi) GetBotConversationMessages(ctx context.Context, req *connect.Request[controlv1.GetBotConversationMessagesRequest]) (*connect.Response[controlv1.GetBotConversationMessagesResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if !au.CanAccessControlPanel() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 200
	}

	rows, err := s.DB.SelectChatBotConversationMessages(protocol, botID, req.Msg.ConversationKey, limit)
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

func (s *ControlApi) StreamBotConversationUpdates(ctx context.Context, req *connect.Request[controlv1.StreamBotConversationUpdatesRequest], stream *connect.ServerStream[controlv1.StreamBotConversationUpdatesResponse]) error {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if !au.CanAccessControlPanel() {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)
	if protocol == "" || botID == "" || strings.TrimSpace(req.Msg.ConversationKey) == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("protocol, bot_id and conversation_key are required"))
	}

	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	cursor := req.Msg.LastMessageId

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			rows, err := s.DB.SelectChatBotConversationMessagesAfterID(protocol, botID, req.Msg.ConversationKey, cursor, 100)
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			if len(rows) == 0 {
				continue
			}
			if err := stream.Send(&controlv1.StreamBotConversationUpdatesResponse{
				NewMessages: marshalBotConversationMessageRows(rows),
			}); err != nil {
				return err
			}
			cursor = rows[len(rows)-1].ID
		}
	}
}

func (s *ControlApi) SendBotConversationMessage(ctx context.Context, req *connect.Request[controlv1.SendBotConversationMessageRequest]) (*connect.Response[controlv1.SendBotConversationMessageResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if !au.CanAccessControlPanel() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}
	if strings.TrimSpace(req.Msg.Content) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is required"))
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Msg.Protocol))
	botID := strings.TrimSpace(req.Msg.BotId)

	channel := ""
	parts := strings.SplitN(req.Msg.ConversationKey, "|", 2)
	if len(parts) > 0 {
		channel = parts[0]
	}
	if channel == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid conversation key"))
	}

	ctrl, err := s.getChatBotControl(protocol, botID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	platformIdentity := ""
	bot := ctrl.GetChatBot()
	if bot != nil {
		platformIdentity = bot.Identity
		if !bot.IsRunning {
			msg := "Bot is offline; cannot send replies"
			if strings.TrimSpace(bot.StatusMessage) != "" {
				msg = fmt.Sprintf("Bot is offline: %s", bot.StatusMessage)
			}
			return connect.NewResponse(&controlv1.SendBotConversationMessageResponse{
				StandardResponse: &controlv1.StandardResponse{Success: false, Message: msg},
			}), nil
		}
	}

	outgoing := &msgs.OutgoingMessage{
		Content:         req.Msg.Content,
		Channel:         channel,
		Protocol:        protocol,
		Identity:        platformIdentity,
		ConversationKey: strings.TrimSpace(req.Msg.ConversationKey),
	}
	amqp.PublishPbWithRoutingKey(outgoing, amqp.GetOutgoingMessageRoutingKey(protocol, platformIdentity))

	return connect.NewResponse(&controlv1.SendBotConversationMessageResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Message sent"},
	}), nil
}
