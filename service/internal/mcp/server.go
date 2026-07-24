// Package mcp provides an MCP (Model Context Protocol) server exposed as an HTTP endpoint.
// It exposes Japella social-account operations as MCP tools, using the same ControlApi
// business layer as the Connect RPC API.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/layers/api"
)

// NewHandler returns an http.Handler that serves the MCP Streamable HTTP endpoint.
// Mount at /mcp. The caller must run Bearer API key auth middleware first so the
// request context contains the authenticated user (via connectrpc authn.SetInfo).
func NewHandler(srv *api.ControlApi) http.Handler {
	mcpServer := server.NewMCPServer(
		"Japella",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	mcpServer.AddTool(mcp.NewTool("japella_list_social_accounts",
		mcp.WithDescription("List social accounts the authenticated user can see. Returns id, connector, identity, active, and posting capabilities."),
		mcp.WithBoolean("only_active", mcp.Description("If true, only return active social accounts. Defaults to false.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListSocialAccounts(ctx, srv, req)
	})

	mcpServer.AddTool(mcp.NewTool("japella_submit_post",
		mcp.WithDescription("Post content to one or more social accounts. Accounts under an AccountPolicy with apply_to_mcp may return state pending_approval instead of posting immediately."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Post body text")),
		mcp.WithArray("social_account_ids", mcp.Required(), mcp.Description("Social account IDs to post to"), mcp.WithNumberItems()),
		mcp.WithString("scheduled_at", mcp.Description("Optional schedule time: RFC3339 or YYYY-MM-DDTHH:MM")),
		mcp.WithNumber("campaign_id", mcp.Description("Optional campaign ID to associate with the post")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSubmitPost(ctx, srv, req)
	})

	mcpServer.AddTool(mcp.NewTool("japella_list_pending_approvals",
		mcp.WithDescription("List posts awaiting the caller's approval at the current approval stage."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListPendingApprovals(ctx, srv)
	})

	mcpServer.AddTool(mcp.NewTool("japella_approve_post",
		mcp.WithDescription("Approve the current stage of a pending post. When all stages are done, the post is published or scheduled."),
		mcp.WithNumber("post_id", mcp.Required(), mcp.Description("Post ID to approve")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleApprovePost(ctx, srv, req)
	})

	mcpServer.AddTool(mcp.NewTool("japella_reject_post",
		mcp.WithDescription("Reject a pending post (current-stage approver or submitter). The post will not be published."),
		mcp.WithNumber("post_id", mcp.Required(), mcp.Description("Post ID to reject")),
		mcp.WithString("reason", mcp.Description("Optional rejection reason")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRejectPost(ctx, srv, req)
	})

	return server.NewStreamableHTTPServer(mcpServer, server.WithEndpointPath("/mcp"))
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func handleListSocialAccounts(ctx context.Context, srv *api.ControlApi, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	onlyActive := req.GetBool("only_active", false)
	res, err := srv.GetSocialAccounts(ctx, connect.NewRequest(&controlv1.GetSocialAccountsRequest{
		OnlyActive: onlyActive,
	}))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accounts := make([]map[string]any, 0, len(res.Msg.GetAccounts()))
	for _, a := range res.Msg.GetAccounts() {
		accounts = append(accounts, map[string]any{
			"id":             a.GetId(),
			"identity":       a.GetIdentity(),
			"connector":      a.GetConnector(),
			"icon":           a.GetIcon(),
			"active":         a.GetActive(),
			"token_expiry":   a.GetTokenExpiry(),
			"owner_user_id":  a.GetOwnerUserId(),
			"owner_username": a.GetOwnerUsername(),
			"is_owner":       a.GetIsOwner(),
			"can_post":       a.GetCanPost(),
			"can_manage":     a.GetCanManage(),
		})
	}
	return jsonResult(map[string]any{"accounts": accounts})
}

func handleSubmitPost(ctx context.Context, srv *api.ControlApi, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ids, err := req.RequireIntSlice("social_account_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(ids) == 0 {
		return mcp.NewToolResultError("social_account_ids must contain at least one account ID"), nil
	}

	socialAccounts := make([]uint32, 0, len(ids))
	for _, id := range ids {
		if id < 0 {
			return mcp.NewToolResultError("social_account_ids must be non-negative"), nil
		}
		socialAccounts = append(socialAccounts, uint32(id))
	}

	ctx = api.WithSubmissionSource(ctx, db.SubmissionSourceMCP)
	submitReq := &controlv1.SubmitPostRequest{
		Content:        content,
		SocialAccounts: socialAccounts,
		ScheduledAt:    req.GetString("scheduled_at", ""),
		CampaignId:     uint32(req.GetInt("campaign_id", 0)),
	}

	res, err := srv.SubmitPost(ctx, connect.NewRequest(submitReq))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	posts := make([]map[string]any, 0, len(res.Msg.GetPosts()))
	for _, p := range res.Msg.GetPosts() {
		posts = append(posts, map[string]any{
			"id":                      p.GetId(),
			"social_account_id":       p.GetSocialAccountId(),
			"social_account_identity": p.GetSocialAccountIdentity(),
			"social_account_icon":     p.GetSocialAccountIcon(),
			"success":                 p.GetSuccess(),
			"post_url":                p.GetPostUrl(),
			"content":                 p.GetContent(),
			"state":                   p.GetState(),
		})
	}
	return jsonResult(map[string]any{"posts": posts})
}

func handleListPendingApprovals(ctx context.Context, srv *api.ControlApi) (*mcp.CallToolResult, error) {
	res, err := srv.ListPendingApprovals(ctx, connect.NewRequest(&controlv1.ListPendingApprovalsRequest{}))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pending := make([]map[string]any, 0, len(res.Msg.GetPending()))
	for _, p := range res.Msg.GetPending() {
		item := map[string]any{
			"approval_stage":        p.GetApprovalStage(),
			"account_policy_id":     p.GetAccountPolicyId(),
			"account_policy_name":   p.GetAccountPolicyName(),
			"submission_source":     p.GetSubmissionSource(),
			"submitted_by_user_id":  p.GetSubmittedByUserId(),
			"submitted_by_username": p.GetSubmittedByUsername(),
			"can_approve":           p.GetCanApprove(),
			"can_reject":            p.GetCanReject(),
			"waiting_on":            p.GetWaitingOn(),
		}
		if post := p.GetPost(); post != nil {
			item["post"] = map[string]any{
				"id":                      post.GetId(),
				"social_account_id":       post.GetSocialAccountId(),
				"social_account_identity": post.GetSocialAccountIdentity(),
				"content":                 post.GetContent(),
				"state":                   post.GetState(),
			}
		}
		pending = append(pending, item)
	}
	return jsonResult(map[string]any{"pending": pending})
}

func handleApprovePost(ctx context.Context, srv *api.ControlApi, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, err := req.RequireInt("post_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if postID < 0 {
		return mcp.NewToolResultError("post_id must be non-negative"), nil
	}
	res, err := srv.ApprovePost(ctx, connect.NewRequest(&controlv1.ApprovePostRequest{PostId: uint32(postID)}))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out := map[string]any{"message": res.Msg.GetStandardResponse().GetMessage()}
	if p := res.Msg.GetPost(); p != nil {
		out["post"] = map[string]any{
			"id":                p.GetId(),
			"social_account_id": p.GetSocialAccountId(),
			"success":           p.GetSuccess(),
			"state":             p.GetState(),
			"post_url":          p.GetPostUrl(),
		}
	}
	return jsonResult(out)
}

func handleRejectPost(ctx context.Context, srv *api.ControlApi, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, err := req.RequireInt("post_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if postID < 0 {
		return mcp.NewToolResultError("post_id must be non-negative"), nil
	}
	res, err := srv.RejectPost(ctx, connect.NewRequest(&controlv1.RejectPostRequest{
		PostId: uint32(postID),
		Reason: req.GetString("reason", ""),
	}))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]any{"message": res.Msg.GetStandardResponse().GetMessage()})
}
