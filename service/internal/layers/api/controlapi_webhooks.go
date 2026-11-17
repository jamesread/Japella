package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/webhook"
)

func webhookTargetToProto(t *db.WebhookTarget) *controlv1.Webhook {
	if t == nil {
		return nil
	}
	created := ""
	updated := ""
	if !t.CreatedAt.IsZero() {
		created = t.CreatedAt.Format(time.RFC3339)
	}
	if !t.UpdatedAt.IsZero() {
		updated = t.UpdatedAt.Format(time.RFC3339)
	}
	return &controlv1.Webhook{
		Id:      t.ID,
		Url:     t.URL,
		Events:  append([]string(nil), t.Events...),
		Enabled: t.Enabled,
		Created: created,
		Updated: updated,
	}
}

func (s *ControlApi) ListWebhooks(ctx context.Context, req *connect.Request[controlv1.ListWebhooksRequest]) (*connect.Response[controlv1.ListWebhooksResponse], error) {
	_ = ctx
	targets, err := s.DB.ListWebhookTargets()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list webhooks"))
	}
	out := make([]*controlv1.Webhook, 0, len(targets))
	for _, t := range targets {
		out = append(out, webhookTargetToProto(t))
	}
	return connect.NewResponse(&controlv1.ListWebhooksResponse{
		Webhooks: out,
		Events:   append([]string(nil), webhook.SupportedEvents...),
	}), nil
}

func webhookDeliveryLogToProto(l *db.WebhookDeliveryLog) *controlv1.WebhookDeliveryLog {
	if l == nil {
		return nil
	}
	created := ""
	if !l.CreatedAt.IsZero() {
		created = l.CreatedAt.Format(time.RFC3339)
	}
	return &controlv1.WebhookDeliveryLog{
		Id:              l.ID,
		WebhookTargetId: l.WebhookTargetID,
		Event:           l.Event,
		Success:         l.Success,
		HttpStatus:      intconv.ToInt32(l.HTTPStatus),
		ErrorMessage:    l.ErrorMessage,
		DurationMs:      intconv.ToInt32(l.DurationMs),
		Created:         created,
	}
}

func (s *ControlApi) GetWebhook(ctx context.Context, req *connect.Request[controlv1.GetWebhookRequest]) (*connect.Response[controlv1.GetWebhookResponse], error) {
	_ = ctx
	existing, err := s.requireWebhookTarget(req.Msg.Id)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.LogLimit)
	if limit == 0 {
		limit = 50
	}
	logs, err := s.DB.ListWebhookDeliveryLogs(req.Msg.Id, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list webhook delivery logs"))
	}
	outLogs := make([]*controlv1.WebhookDeliveryLog, 0, len(logs))
	for _, l := range logs {
		outLogs = append(outLogs, webhookDeliveryLogToProto(l))
	}
	return connect.NewResponse(&controlv1.GetWebhookResponse{
		Webhook:      webhookTargetToProto(existing),
		DeliveryLogs: outLogs,
	}), nil
}

func (s *ControlApi) requireWebhookTarget(id uint32) (*db.WebhookTarget, error) {
	if id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook id is required"))
	}
	existing, err := s.DB.GetWebhookTarget(id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load webhook"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("webhook not found"))
	}
	return existing, nil
}

func (s *ControlApi) CreateWebhook(ctx context.Context, req *connect.Request[controlv1.CreateWebhookRequest]) (*connect.Response[controlv1.CreateWebhookResponse], error) {
	_ = ctx
	url, err := webhook.NormalizeURL(req.Msg.Url)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	secret := strings.TrimSpace(req.Msg.Secret)
	if secret == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook secret is required"))
	}
	events, err := webhook.NormalizeEvents(req.Msg.Events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	id, err := s.DB.CreateWebhookTarget(url, secret, events, req.Msg.Enabled)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create webhook"))
	}
	return connect.NewResponse(&controlv1.CreateWebhookResponse{Id: id}), nil
}

func (s *ControlApi) updateWebhookTargetFields(id uint32, existing *db.WebhookTarget, url string, events []string, enabled bool, secret string) error {
	if secret == "" {
		return s.DB.UpdateWebhookTarget(id, url, existing.Secret, events, enabled, true)
	}
	return s.DB.UpdateWebhookTarget(id, url, secret, events, enabled, false)
}

func (s *ControlApi) UpdateWebhook(ctx context.Context, req *connect.Request[controlv1.UpdateWebhookRequest]) (*connect.Response[controlv1.UpdateWebhookResponse], error) {
	_ = ctx
	existing, err := s.requireWebhookTarget(req.Msg.Id)
	if err != nil {
		return nil, err
	}
	url, err := webhook.NormalizeURL(req.Msg.Url)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	events, err := webhook.NormalizeEvents(req.Msg.Events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.updateWebhookTargetFields(req.Msg.Id, existing, url, events, req.Msg.Enabled, strings.TrimSpace(req.Msg.Secret)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update webhook"))
	}
	return connect.NewResponse(&controlv1.UpdateWebhookResponse{}), nil
}

func (s *ControlApi) DeleteWebhook(ctx context.Context, req *connect.Request[controlv1.DeleteWebhookRequest]) (*connect.Response[controlv1.DeleteWebhookResponse], error) {
	_ = ctx
	if _, err := s.requireWebhookTarget(req.Msg.Id); err != nil {
		return nil, err
	}
	if err := s.DB.DeleteWebhookTarget(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete webhook"))
	}
	return connect.NewResponse(&controlv1.DeleteWebhookResponse{}), nil
}

func (s *ControlApi) TestWebhook(ctx context.Context, req *connect.Request[controlv1.TestWebhookRequest]) (*connect.Response[controlv1.TestWebhookResponse], error) {
	existing, err := s.requireWebhookTarget(req.Msg.Id)
	if err != nil {
		return nil, err
	}

	deliverCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	status, err := s.webhookDispatcher().DeliverOne(deliverCtx, webhook.Target{
		ID:     existing.ID,
		URL:    existing.URL,
		Secret: existing.Secret,
	}, webhook.EventWebhookTest, testWebhookPayload(existing))
	if err != nil {
		return connect.NewResponse(&controlv1.TestWebhookResponse{
			Success:    false,
			Message:    err.Error(),
			HttpStatus: 0,
		}), nil
	}
	return connect.NewResponse(testWebhookResponse(status)), nil
}

func testWebhookPayload(existing *db.WebhookTarget) map[string]any {
	return map[string]any{
		"test": map[string]any{
			"message": "This is a Japella webhook.test event.",
			"webhook": map[string]any{
				"id":  existing.ID,
				"url": existing.URL,
			},
		},
	}
}

func testWebhookResponse(status int) *controlv1.TestWebhookResponse {
	ok := status >= 200 && status < 300
	msg := fmt.Sprintf("Endpoint returned HTTP %d", status)
	if !ok {
		msg = fmt.Sprintf("Endpoint returned non-success HTTP %d", status)
	}
	return &controlv1.TestWebhookResponse{
		Success:    ok,
		Message:    msg,
		HttpStatus: intconv.ToInt32(status),
	}
}

func (s *ControlApi) webhookDispatcher() *webhook.Dispatcher {
	return webhook.NewDispatcher(s.DB)
}

func (s *ControlApi) buildPostWebhookPayload(post *db.Post, socialAccount *db.SocialAccount, state, postURL string) map[string]any {
	postMap := map[string]any{
		"id":      post.ID,
		"content": post.Content,
		"state":   state,
	}
	if post.SocialAccountID.Valid {
		postMap["social_account_id"] = post.SocialAccountIDUint()
	}
	if postURL != "" {
		postMap["post_url"] = postURL
	}
	if socialAccount != nil {
		postMap["social_account_identity"] = socialAccount.Identity
		postMap["connector"] = socialAccount.Connector
	}
	if post.CampaignID.Valid {
		postMap["campaign_id"] = post.CampaignID.Int32
	}
	return map[string]any{"post": postMap}
}

func (s *ControlApi) dispatchPostOutcome(ctx context.Context, post *db.Post, socialAccount *db.SocialAccount, success bool, postURL string) {
	if post == nil {
		return
	}
	if success {
		s.webhookDispatcher().Dispatch(ctx, webhook.EventPostCompleted, s.buildPostWebhookPayload(post, socialAccount, db.PostStateCompleted, postURL))
		return
	}
	s.webhookDispatcher().Dispatch(ctx, webhook.EventPostError, s.buildPostWebhookPayload(post, socialAccount, db.PostStateError, postURL))
}

func (s *ControlApi) dispatchApprovalRequested(ctx context.Context, post *db.Post, socialAccount *db.SocialAccount, policy *db.AccountPolicy) {
	if post == nil {
		return
	}
	postPayload := map[string]any{
		"id":             post.ID,
		"content":        post.Content,
		"state":          post.State,
		"approval_stage": post.ApprovalStage,
	}
	if post.SocialAccountID.Valid {
		postPayload["social_account_id"] = post.SocialAccountIDUint()
	}
	payload := map[string]any{
		"post": postPayload,
	}
	if socialAccount != nil {
		payload["post"].(map[string]any)["social_account_identity"] = socialAccount.Identity
		payload["post"].(map[string]any)["connector"] = socialAccount.Connector
	}
	if policy != nil {
		payload["post"].(map[string]any)["account_policy_id"] = policy.ID
		payload["post"].(map[string]any)["account_policy_name"] = policy.Name
	}
	s.webhookDispatcher().Dispatch(ctx, webhook.EventApprovalRequested, payload)
}

func (s *ControlApi) botDisplayNameByKey() map[string]string {
	nameByBot := map[string]string{}
	instances, err := s.DB.SelectChatBotInstances()
	if err != nil {
		return nameByBot
	}
	for _, inst := range instances {
		name := strings.TrimSpace(inst.DisplayName)
		if name == "" {
			name = inst.BotID
		}
		nameByBot[inst.Protocol+"\x00"+inst.BotID] = name
	}
	return nameByBot
}

func botWebhookDisplayName(nameByBot map[string]string, protocol, botID, identity string) string {
	if name := nameByBot[protocol+"\x00"+botID]; name != "" {
		return name
	}
	if identity != "" {
		return identity
	}
	return botID
}

func botWebhookToProto(hook *db.WebhookHook, nameByBot map[string]string) *controlv1.BotWebhook {
	protocol := strings.TrimSpace(hook.Connector)
	botID := strings.TrimSpace(hook.BotID)
	return &controlv1.BotWebhook{
		Id:       hook.ID,
		Protocol: protocol,
		BotId:    botID,
		BotName:  botWebhookDisplayName(nameByBot, protocol, botID, hook.Identity),
		Url:      hook.URL,
		Enabled:  hook.Enabled,
	}
}

func (s *ControlApi) ListBotWebhooks(ctx context.Context, req *connect.Request[controlv1.ListBotWebhooksRequest]) (*connect.Response[controlv1.ListBotWebhooksResponse], error) {
	_ = ctx
	_ = s.DB.DeleteOrphanedWebhookHooks()

	hooks, err := s.DB.ListWebhookHooksForExistingBots()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list bot webhooks"))
	}

	nameByBot := s.botDisplayNameByKey()
	out := make([]*controlv1.BotWebhook, 0, len(hooks))
	for _, hook := range hooks {
		out = append(out, botWebhookToProto(hook, nameByBot))
	}
	return connect.NewResponse(&controlv1.ListBotWebhooksResponse{Webhooks: out}), nil
}
