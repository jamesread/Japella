package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/llm"
	"github.com/jamesread/japella/internal/rbac"
	log "github.com/sirupsen/logrus"
)

func (s *ControlApi) copyLlmToolRunsFromDraft(fromDraftID, toPostID uint32) {
	if fromDraftID == 0 || toPostID == 0 || fromDraftID == toPostID {
		return
	}
	if err := s.DB.CopyLlmToolRuns(fromDraftID, toPostID); err != nil {
		log.Warnf("Failed to copy LLM tool runs from draft %d to post %d: %v", fromDraftID, toPostID, err)
	}
}

func llmConnectionToProto(c *db.LlmConnection) *controlv1.LlmConnection {
	if c == nil {
		return nil
	}
	created := ""
	updated := ""
	if !c.CreatedAt.IsZero() {
		created = c.CreatedAt.Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		updated = c.UpdatedAt.Format(time.RFC3339)
	}
	return &controlv1.LlmConnection{
		Id:        c.ID,
		Name:      c.Name,
		BaseUrl:   c.BaseURL,
		ApiKeySet: strings.TrimSpace(c.APIKey) != "",
		Model:     c.ModelName,
		Enabled:   c.Enabled,
		Created:   created,
		Updated:   updated,
	}
}

func llmToolToProto(t *db.LlmTool) *controlv1.LlmTool {
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
	return &controlv1.LlmTool{
		Id:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		LlmConnectionId: t.LlmConnectionID,
		ConnectionName:  t.ConnectionName,
		Prompt:          t.Prompt,
		Enabled:         t.Enabled,
		SortOrder:       t.SortOrder,
		Created:         created,
		Updated:         updated,
	}
}

func llmToolRunToProto(r *db.LlmToolRun) *controlv1.LlmToolRun {
	if r == nil {
		return nil
	}
	created := ""
	if r.CreatedAt.Valid {
		created = r.CreatedAt.Time.Format(time.RFC3339)
	}
	var toolID uint32
	if r.LlmToolID.Valid {
		toolID = uint32(r.LlmToolID.Int32)
	}
	var createdBy uint32
	if r.CreatedByUserID.Valid {
		createdBy = uint32(r.CreatedByUserID.Int32)
	}
	return &controlv1.LlmToolRun{
		Id:              r.ID,
		PostId:          r.PostID,
		LlmToolId:       toolID,
		ToolName:        r.ToolName,
		ConnectionName:  r.ConnectionName,
		Model:           r.ModelName,
		InputContent:    r.InputContent,
		OutputContent:   r.OutputContent,
		Status:          r.Status,
		ErrorMessage:    r.ErrorMessage,
		CreatedByUserId: createdBy,
		Created:         created,
	}
}

func (s *ControlApi) ListLlmConnections(ctx context.Context, req *connect.Request[controlv1.ListLlmConnectionsRequest]) (*connect.Response[controlv1.ListLlmConnectionsResponse], error) {
	_ = ctx
	_ = req
	rows, err := s.DB.ListLlmConnections()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list llm connections"))
	}
	out := make([]*controlv1.LlmConnection, 0, len(rows))
	for _, c := range rows {
		out = append(out, llmConnectionToProto(c))
	}
	return connect.NewResponse(&controlv1.ListLlmConnectionsResponse{Connections: out}), nil
}

func (s *ControlApi) GetLlmConnection(ctx context.Context, req *connect.Request[controlv1.GetLlmConnectionRequest]) (*connect.Response[controlv1.GetLlmConnectionResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	c, err := s.DB.GetLlmConnection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if c == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm connection not found"))
	}
	return connect.NewResponse(&controlv1.GetLlmConnectionResponse{Connection: llmConnectionToProto(c)}), nil
}

func (s *ControlApi) CreateLlmConnection(ctx context.Context, req *connect.Request[controlv1.CreateLlmConnectionRequest]) (*connect.Response[controlv1.CreateLlmConnectionResponse], error) {
	_ = ctx
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	baseURL, err := llm.NormalizeBaseURL(req.Msg.BaseUrl)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	model := strings.TrimSpace(req.Msg.Model)
	if model == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model is required"))
	}
	id, err := s.DB.CreateLlmConnection(name, baseURL, strings.TrimSpace(req.Msg.ApiKey), model, req.Msg.Enabled)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create llm connection"))
	}
	return connect.NewResponse(&controlv1.CreateLlmConnectionResponse{Id: id}), nil
}

func (s *ControlApi) UpdateLlmConnection(ctx context.Context, req *connect.Request[controlv1.UpdateLlmConnectionRequest]) (*connect.Response[controlv1.UpdateLlmConnectionResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetLlmConnection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm connection not found"))
	}
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	baseURL, err := llm.NormalizeBaseURL(req.Msg.BaseUrl)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	model := strings.TrimSpace(req.Msg.Model)
	if model == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model is required"))
	}
	apiKey := strings.TrimSpace(req.Msg.ApiKey)
	keepKey := apiKey == ""
	if err := s.DB.UpdateLlmConnection(req.Msg.Id, name, baseURL, apiKey, model, req.Msg.Enabled, keepKey); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update llm connection"))
	}
	return connect.NewResponse(&controlv1.UpdateLlmConnectionResponse{}), nil
}

func (s *ControlApi) DeleteLlmConnection(ctx context.Context, req *connect.Request[controlv1.DeleteLlmConnectionRequest]) (*connect.Response[controlv1.DeleteLlmConnectionResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetLlmConnection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm connection not found"))
	}
	count, err := s.DB.CountLlmToolsForConnection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check llm tools"))
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connection is used by %d tool(s); delete or reassign them first", count))
	}
	if err := s.DB.DeleteLlmConnection(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete llm connection"))
	}
	return connect.NewResponse(&controlv1.DeleteLlmConnectionResponse{}), nil
}

func (s *ControlApi) TestLlmConnection(ctx context.Context, req *connect.Request[controlv1.TestLlmConnectionRequest]) (*connect.Response[controlv1.TestLlmConnectionResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	c, err := s.DB.GetLlmConnection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if c == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm connection not found"))
	}
	client := llm.NewClient()
	if err := client.Ping(ctx, c.BaseURL, c.APIKey, c.ModelName); err != nil {
		log.Warnf("TestLlmConnection id=%d name=%q failed: %v", c.ID, c.Name, err)
		return connect.NewResponse(&controlv1.TestLlmConnectionResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}
	log.Infof("TestLlmConnection id=%d name=%q ok", c.ID, c.Name)
	return connect.NewResponse(&controlv1.TestLlmConnectionResponse{
		Success: true,
		Message: "Connection OK",
	}), nil
}

func (s *ControlApi) ListLlmConnectionPresets(ctx context.Context, req *connect.Request[controlv1.ListLlmConnectionPresetsRequest]) (*connect.Response[controlv1.ListLlmConnectionPresetsResponse], error) {
	_ = ctx
	_ = req
	out := make([]*controlv1.LlmConnectionPreset, 0, len(llm.ConnectionPresets))
	for _, p := range llm.ConnectionPresets {
		out = append(out, &controlv1.LlmConnectionPreset{
			Id:             p.ID,
			Name:           p.Name,
			BaseUrl:        p.BaseURL,
			SuggestedModel: p.SuggestedModel,
			Description:    p.Description,
		})
	}
	return connect.NewResponse(&controlv1.ListLlmConnectionPresetsResponse{Presets: out}), nil
}

func (s *ControlApi) ListLlmConnectionModels(ctx context.Context, req *connect.Request[controlv1.ListLlmConnectionModelsRequest]) (*connect.Response[controlv1.ListLlmConnectionModelsResponse], error) {
	baseURL := strings.TrimSpace(req.Msg.BaseUrl)
	apiKey := strings.TrimSpace(req.Msg.ApiKey)

	if req.Msg.ConnectionId != 0 {
		c, err := s.DB.GetLlmConnection(req.Msg.ConnectionId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
		}
		if c == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm connection not found"))
		}
		if baseURL == "" {
			baseURL = c.BaseURL
		}
		if apiKey == "" {
			apiKey = c.APIKey
		}
	}
	if baseURL == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("base_url or connection_id is required"))
	}
	normalized, err := llm.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	client := llm.NewClient()
	models, err := client.ListModels(ctx, normalized, apiKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := make([]*controlv1.LlmModel, 0, len(models))
	for _, m := range models {
		out = append(out, &controlv1.LlmModel{Id: m.ID, Name: m.Name})
	}
	return connect.NewResponse(&controlv1.ListLlmConnectionModelsResponse{Models: out}), nil
}

func (s *ControlApi) ListLlmTools(ctx context.Context, req *connect.Request[controlv1.ListLlmToolsRequest]) (*connect.Response[controlv1.ListLlmToolsResponse], error) {
	_ = ctx
	_ = req
	rows, err := s.DB.ListLlmTools()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list llm tools"))
	}
	out := make([]*controlv1.LlmTool, 0, len(rows))
	for _, t := range rows {
		out = append(out, llmToolToProto(t))
	}
	return connect.NewResponse(&controlv1.ListLlmToolsResponse{Tools: out}), nil
}

func (s *ControlApi) GetLlmTool(ctx context.Context, req *connect.Request[controlv1.GetLlmToolRequest]) (*connect.Response[controlv1.GetLlmToolResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	t, err := s.DB.GetLlmTool(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm tool"))
	}
	if t == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm tool not found"))
	}
	return connect.NewResponse(&controlv1.GetLlmToolResponse{Tool: llmToolToProto(t)}), nil
}

func (s *ControlApi) CreateLlmTool(ctx context.Context, req *connect.Request[controlv1.CreateLlmToolRequest]) (*connect.Response[controlv1.CreateLlmToolResponse], error) {
	_ = ctx
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	prompt := strings.TrimSpace(req.Msg.Prompt)
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prompt is required"))
	}
	if req.Msg.LlmConnectionId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("llm_connection_id is required"))
	}
	conn, err := s.DB.GetLlmConnection(req.Msg.LlmConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if conn == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("llm connection not found"))
	}
	id, err := s.DB.CreateLlmTool(name, strings.TrimSpace(req.Msg.Description), req.Msg.LlmConnectionId, prompt, req.Msg.Enabled, req.Msg.SortOrder)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create llm tool"))
	}
	return connect.NewResponse(&controlv1.CreateLlmToolResponse{Id: id}), nil
}

func (s *ControlApi) UpdateLlmTool(ctx context.Context, req *connect.Request[controlv1.UpdateLlmToolRequest]) (*connect.Response[controlv1.UpdateLlmToolResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetLlmTool(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm tool"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm tool not found"))
	}
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	prompt := strings.TrimSpace(req.Msg.Prompt)
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prompt is required"))
	}
	if req.Msg.LlmConnectionId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("llm_connection_id is required"))
	}
	conn, err := s.DB.GetLlmConnection(req.Msg.LlmConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if conn == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("llm connection not found"))
	}
	if err := s.DB.UpdateLlmTool(req.Msg.Id, name, strings.TrimSpace(req.Msg.Description), req.Msg.LlmConnectionId, prompt, req.Msg.Enabled, req.Msg.SortOrder); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update llm tool"))
	}
	return connect.NewResponse(&controlv1.UpdateLlmToolResponse{}), nil
}

func (s *ControlApi) DeleteLlmTool(ctx context.Context, req *connect.Request[controlv1.DeleteLlmToolRequest]) (*connect.Response[controlv1.DeleteLlmToolResponse], error) {
	_ = ctx
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetLlmTool(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm tool"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm tool not found"))
	}
	if err := s.DB.DeleteLlmTool(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete llm tool"))
	}
	return connect.NewResponse(&controlv1.DeleteLlmToolResponse{}), nil
}

func (s *ControlApi) ListEnabledLlmTools(ctx context.Context, req *connect.Request[controlv1.ListEnabledLlmToolsRequest]) (*connect.Response[controlv1.ListEnabledLlmToolsResponse], error) {
	_ = ctx
	_ = req
	rows, err := s.DB.ListEnabledLlmTools()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list enabled llm tools"))
	}
	out := make([]*controlv1.LlmToolSummary, 0, len(rows))
	for _, t := range rows {
		out = append(out, &controlv1.LlmToolSummary{
			Id:          t.ID,
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return connect.NewResponse(&controlv1.ListEnabledLlmToolsResponse{Tools: out}), nil
}

func (s *ControlApi) canEditPostContent(auUserID uint32, post *db.Post) bool {
	if post == nil {
		return false
	}
	isSubmitter := post.SubmittedByUserID.Valid && uint32(post.SubmittedByUserID.Int32) == auUserID
	switch post.State {
	case db.PostStateDraft:
		return isSubmitter
	case db.PostStatePendingApproval:
		if isSubmitter {
			return true
		}
		if post.AccountPolicyID.Valid {
			stage, stErr := s.DB.GetApprovalStageByPolicyAndOrder(uint32(post.AccountPolicyID.Int32), post.ApprovalStage)
			if stErr == nil {
				ok, _ := s.DB.CanUserApproveStage(auUserID, stage)
				return ok
			}
		}
		return false
	default:
		return false
	}
}

func (s *ControlApi) canViewPostForLlm(au *authentication.AuthenticatedUser, post *db.Post) bool {
	if au == nil || au.User == nil || post == nil {
		return false
	}
	if au.HasPermission(rbac.PermissionSocialAccountsViewAll) || (au.RBAC != nil && au.RBAC.IsSuperuser) {
		return true
	}
	if post.SubmittedByUserID.Valid && uint32(post.SubmittedByUserID.Int32) == au.User.ID {
		return true
	}
	if post.State == db.PostStatePendingApproval {
		pending := s.marshalPendingApproval(post, au.User.ID)
		if pending.CanApprove || pending.CanReject {
			return true
		}
		// Visible via list pending (view-all / manage policies already covered above).
		if au.HasPermission(rbac.PermissionAccountPoliciesManage) {
			return true
		}
	}
	if post.SocialAccountID.Valid {
		return s.DB.CanUserAccessSocialAccount(au.User.ID, uint32(post.SocialAccountID.Int32), "read")
	}
	return false
}

func (s *ControlApi) UpdatePostContent(ctx context.Context, req *connect.Request[controlv1.UpdatePostContentRequest]) (*connect.Response[controlv1.UpdatePostContentResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if req.Msg.PostId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id is required"))
	}
	content := strings.TrimSpace(req.Msg.Content)
	if content == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is required"))
	}

	post, err := s.DB.GetPost(req.Msg.PostId)
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	if post.State != db.PostStateDraft && post.State != db.PostStatePendingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post content can only be updated for draft or pending approval posts"))
	}
	if !s.canEditPostContent(au.User.ID, post) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to edit this post"))
	}
	if err := s.DB.UpdatePostContent(post.ID, content); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update post content"))
	}
	post.Content = content
	log.Infof("Post %d content updated by %s (state=%s)", post.ID, au.User.Username, post.State)
	return connect.NewResponse(&controlv1.UpdatePostContentResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post content saved"},
		Post:             s.marshalPostStatus(post),
	}), nil
}

func (s *ControlApi) RunLlmTool(ctx context.Context, req *connect.Request[controlv1.RunLlmToolRequest]) (*connect.Response[controlv1.RunLlmToolResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if req.Msg.PostId == 0 || req.Msg.ToolId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id and tool_id are required"))
	}

	post, err := s.DB.GetPost(req.Msg.PostId)
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	if !s.canEditPostContent(au.User.ID, post) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to run tools on this post"))
	}
	content := strings.TrimSpace(post.Content)
	if content == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post content is empty"))
	}

	tool, err := s.DB.GetLlmTool(req.Msg.ToolId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm tool"))
	}
	if tool == nil || !tool.Enabled {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("llm tool not found"))
	}
	conn, err := s.DB.GetLlmConnection(tool.LlmConnectionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load llm connection"))
	}
	if conn == nil || !conn.Enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("llm connection is not available"))
	}

	client := llm.NewClient()
	output, chatErr := client.ChatCompletion(ctx, conn.BaseURL, conn.APIKey, conn.ModelName, []llm.ChatMessage{
		{Role: "system", Content: tool.Prompt},
		{Role: "user", Content: content},
	})

	run := &db.LlmToolRun{
		PostID:          post.ID,
		LlmToolID:       sql.NullInt32{Int32: int32(tool.ID), Valid: true},
		ToolName:        tool.Name,
		ConnectionName:  conn.Name,
		ModelName:       conn.ModelName,
		InputContent:    content,
		CreatedByUserID: sql.NullInt32{Int32: int32(au.User.ID), Valid: true},
	}
	if chatErr != nil {
		run.Status = db.LlmToolRunStatusError
		run.ErrorMessage = chatErr.Error()
		run.OutputContent = ""
		log.Warnf("RunLlmTool post=%d tool=%q connection=%q failed: %v", post.ID, tool.Name, conn.Name, chatErr)
	} else {
		run.Status = db.LlmToolRunStatusOK
		run.OutputContent = output
		log.Infof("RunLlmTool post=%d tool=%q connection=%q ok", post.ID, tool.Name, conn.Name)
	}

	runID, err := s.DB.CreateLlmToolRun(run)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save llm tool run"))
	}
	run.ID = runID
	run.CreatedAt = sql.NullTime{Time: time.Now(), Valid: true}

	return connect.NewResponse(&controlv1.RunLlmToolResponse{Run: llmToolRunToProto(run)}), nil
}

func (s *ControlApi) ListLlmToolRuns(ctx context.Context, req *connect.Request[controlv1.ListLlmToolRunsRequest]) (*connect.Response[controlv1.ListLlmToolRunsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if req.Msg.PostId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id is required"))
	}
	post, err := s.DB.GetPost(req.Msg.PostId)
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	if !s.canViewPostForLlm(au, post) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to view this post"))
	}
	rows, err := s.DB.ListLlmToolRunsByPost(req.Msg.PostId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list llm tool runs"))
	}
	out := make([]*controlv1.LlmToolRun, 0, len(rows))
	for _, r := range rows {
		out = append(out, llmToolRunToProto(r))
	}
	return connect.NewResponse(&controlv1.ListLlmToolRunsResponse{Runs: out}), nil
}
