package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
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

func (s *ControlApi) UpdateWebhook(ctx context.Context, req *connect.Request[controlv1.UpdateWebhookRequest]) (*connect.Response[controlv1.UpdateWebhookResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook id is required"))
	}
	existing, err := s.DB.GetWebhookTarget(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load webhook"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("webhook not found"))
	}
	url, err := webhook.NormalizeURL(req.Msg.Url)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	events, err := webhook.NormalizeEvents(req.Msg.Events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	secret := strings.TrimSpace(req.Msg.Secret)
	keepSecret := secret == ""
	if keepSecret {
		err = s.DB.UpdateWebhookTarget(req.Msg.Id, url, existing.Secret, events, req.Msg.Enabled, true)
	} else {
		err = s.DB.UpdateWebhookTarget(req.Msg.Id, url, secret, events, req.Msg.Enabled, false)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update webhook"))
	}
	return connect.NewResponse(&controlv1.UpdateWebhookResponse{}), nil
}

func (s *ControlApi) DeleteWebhook(ctx context.Context, req *connect.Request[controlv1.DeleteWebhookRequest]) (*connect.Response[controlv1.DeleteWebhookResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook id is required"))
	}
	existing, err := s.DB.GetWebhookTarget(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load webhook"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("webhook not found"))
	}
	if err := s.DB.DeleteWebhookTarget(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete webhook"))
	}
	return connect.NewResponse(&controlv1.DeleteWebhookResponse{}), nil
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

func (s *ControlApi) ListBotWebhooks(ctx context.Context, req *connect.Request[controlv1.ListBotWebhooksRequest]) (*connect.Response[controlv1.ListBotWebhooksResponse], error) {
	_ = ctx
	_ = s.DB.DeleteOrphanedWebhookHooks()

	hooks, err := s.DB.ListWebhookHooksForExistingBots()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list bot webhooks"))
	}

	nameByBot := map[string]string{}
	instances, err := s.DB.SelectChatBotInstances()
	if err == nil {
		for _, inst := range instances {
			key := inst.Protocol + "\x00" + inst.BotID
			name := strings.TrimSpace(inst.DisplayName)
			if name == "" {
				name = inst.BotID
			}
			nameByBot[key] = name
		}
	}

	out := make([]*controlv1.BotWebhook, 0, len(hooks))
	for _, hook := range hooks {
		protocol := strings.TrimSpace(hook.Connector)
		botID := strings.TrimSpace(hook.BotID)
		key := protocol + "\x00" + botID
		botName := nameByBot[key]
		if botName == "" {
			botName = hook.Identity
		}
		if botName == "" {
			botName = botID
		}
		out = append(out, &controlv1.BotWebhook{
			Id:       hook.ID,
			Protocol: protocol,
			BotId:    botID,
			BotName:  botName,
			Url:      hook.URL,
			Enabled:  hook.Enabled,
		})
	}
	return connect.NewResponse(&controlv1.ListBotWebhooksResponse{Webhooks: out}), nil
}
