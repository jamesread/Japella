package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/rbac"
	log "github.com/sirupsen/logrus"
)

func (s *ControlApi) marshalAccountPolicy(policy *db.AccountPolicy) (*controlv1.AccountPolicy, error) {
	stages, err := s.DB.GetAccountPolicyApprovalStages(policy.ID)
	if err != nil {
		return nil, err
	}
	accountIDs, err := s.DB.GetAccountPolicySocialAccountIDs(policy.ID)
	if err != nil {
		return nil, err
	}

	out := &controlv1.AccountPolicy{
		Id:               policy.ID,
		Name:             policy.Name,
		Description:      policy.Description,
		ApplyToMcp:       policy.ApplyToMCP,
		ApplyToUi:        policy.ApplyToUI,
		SocialAccountIds: accountIDs,
		Stages:           make([]*controlv1.AccountPolicyApprovalStage, 0, len(stages)),
	}
	for _, st := range stages {
		ps := &controlv1.AccountPolicyApprovalStage{
			Id:         st.ID,
			StageOrder: st.StageOrder,
		}
		if st.UserID.Valid {
			ps.UserId = uint32(st.UserID.Int32)
		}
		if st.UserGroupID.Valid {
			ps.UserGroupId = uint32(st.UserGroupID.Int32)
		}
		if st.Username.Valid {
			ps.Username = st.Username.String
		}
		if st.UserGroupName.Valid {
			ps.UserGroupName = st.UserGroupName.String
		}
		out.Stages = append(out.Stages, ps)
	}
	return out, nil
}

func validatePolicyStages(stages []*controlv1.AccountPolicyApprovalStage) ([]*db.AccountPolicyApprovalStage, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("at least one approval stage is required")
	}
	out := make([]*db.AccountPolicyApprovalStage, 0, len(stages))
	for i, st := range stages {
		hasUser := st.UserId != 0
		hasGroup := st.UserGroupId != 0
		if hasUser == hasGroup {
			return nil, fmt.Errorf("stage %d must set exactly one of user_id or user_group_id", i)
		}
		dbStage := &db.AccountPolicyApprovalStage{StageOrder: uint32(i)}
		if hasUser {
			dbStage.UserID = sql.NullInt32{Int32: int32(st.UserId), Valid: true}
		} else {
			dbStage.UserGroupID = sql.NullInt32{Int32: int32(st.UserGroupId), Valid: true}
		}
		out = append(out, dbStage)
	}
	return out, nil
}

func (s *ControlApi) ListAccountPolicies(ctx context.Context, req *connect.Request[controlv1.ListAccountPoliciesRequest]) (*connect.Response[controlv1.ListAccountPoliciesResponse], error) {
	policies, err := s.DB.SelectAccountPolicies()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list account policies"))
	}
	out := make([]*controlv1.AccountPolicy, 0, len(policies))
	for _, p := range policies {
		mp, err := s.marshalAccountPolicy(p)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load policy details"))
		}
		out = append(out, mp)
	}
	return connect.NewResponse(&controlv1.ListAccountPoliciesResponse{Policies: out}), nil
}

func (s *ControlApi) GetAccountPolicy(ctx context.Context, req *connect.Request[controlv1.GetAccountPolicyRequest]) (*connect.Response[controlv1.GetAccountPolicyResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	policy, err := s.DB.GetAccountPolicy(req.Msg.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("policy not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get policy"))
	}
	mp, err := s.marshalAccountPolicy(policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load policy details"))
	}
	return connect.NewResponse(&controlv1.GetAccountPolicyResponse{Policy: mp}), nil
}

func (s *ControlApi) CreateAccountPolicy(ctx context.Context, req *connect.Request[controlv1.CreateAccountPolicyRequest]) (*connect.Response[controlv1.CreateAccountPolicyResponse], error) {
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	stages, err := validatePolicyStages(req.Msg.Stages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	id, err := s.DB.CreateAccountPolicy(&db.AccountPolicy{
		Name:        name,
		Description: req.Msg.Description,
		ApplyToMCP:  req.Msg.ApplyToMcp,
		ApplyToUI:   req.Msg.ApplyToUi,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("could not create policy (duplicate name?)"))
	}
	if err := s.DB.SetAccountPolicyApprovalStages(id, stages); err != nil {
		_ = s.DB.DeleteAccountPolicy(id)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save stages: %w", err))
	}
	if err := s.DB.SetAccountPolicySocialAccounts(id, req.Msg.SocialAccountIds); err != nil {
		_ = s.DB.DeleteAccountPolicy(id)
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to attach social accounts (already on another policy?)"))
	}
	return connect.NewResponse(&controlv1.CreateAccountPolicyResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Policy created"},
		PolicyId:         id,
	}), nil
}

func (s *ControlApi) UpdateAccountPolicy(ctx context.Context, req *connect.Request[controlv1.UpdateAccountPolicyRequest]) (*connect.Response[controlv1.UpdateAccountPolicyResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	stages, err := validatePolicyStages(req.Msg.Stages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := s.DB.GetAccountPolicy(req.Msg.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("policy not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get policy"))
	}

	if err := s.DB.UpdateAccountPolicy(&db.AccountPolicy{
		Model:       db.Model{ID: req.Msg.Id},
		Name:        name,
		Description: req.Msg.Description,
		ApplyToMCP:  req.Msg.ApplyToMcp,
		ApplyToUI:   req.Msg.ApplyToUi,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update policy"))
	}
	if err := s.DB.SetAccountPolicyApprovalStages(req.Msg.Id, stages); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save stages: %w", err))
	}
	if err := s.DB.SetAccountPolicySocialAccounts(req.Msg.Id, req.Msg.SocialAccountIds); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to attach social accounts (already on another policy?)"))
	}
	return connect.NewResponse(&controlv1.UpdateAccountPolicyResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Policy updated"},
	}), nil
}

func (s *ControlApi) DeleteAccountPolicy(ctx context.Context, req *connect.Request[controlv1.DeleteAccountPolicyRequest]) (*connect.Response[controlv1.DeleteAccountPolicyResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	if err := s.DB.DeleteAccountPolicy(req.Msg.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("policy not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete policy"))
	}
	return connect.NewResponse(&controlv1.DeleteAccountPolicyResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Policy deleted"},
	}), nil
}

func (s *ControlApi) ListPendingApprovals(ctx context.Context, req *connect.Request[controlv1.ListPendingApprovalsRequest]) (*connect.Response[controlv1.ListPendingApprovalsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	viewAll := au.HasPermission(rbac.PermissionAccountPoliciesManage) ||
		au.HasPermission(rbac.PermissionSocialAccountsViewAll) ||
		(au.RBAC != nil && au.RBAC.IsSuperuser)

	posts, err := s.DB.SelectPendingApprovalsVisibleToUser(au.User.ID, viewAll)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list pending approvals"))
	}

	out := make([]*controlv1.PendingApproval, 0, len(posts))
	for _, p := range posts {
		item := s.marshalPendingApproval(p, au.User.ID)
		out = append(out, item)
	}
	return connect.NewResponse(&controlv1.ListPendingApprovalsResponse{Pending: out}), nil
}

func (s *ControlApi) marshalPendingApproval(p *db.Post, viewerID uint32) *controlv1.PendingApproval {
	postedDate := p.CreatedAt
	if p.ScheduledAt.Valid {
		postedDate = p.ScheduledAt.Time
	}

	item := &controlv1.PendingApproval{
		Post: &controlv1.PostStatus{
			Id:              p.ID,
			SocialAccountId: p.SocialAccountID,
			Content:         p.Content,
			State:           p.State,
			Success:         false,
			PostUrl:         p.PostURL.String,
			Created:         p.CreatedAt.Format("2006-01-02 15:04:05"),
			PostedDate:      postedDate.Format("2006-01-02 15:04:05"),
		},
		ApprovalStage:    p.ApprovalStage,
		SubmissionSource: p.SubmissionSource,
	}
	if p.AccountPolicyID.Valid {
		item.AccountPolicyId = uint32(p.AccountPolicyID.Int32)
		if pol, err := s.DB.GetAccountPolicy(item.AccountPolicyId); err == nil && pol != nil {
			item.AccountPolicyName = pol.Name
		}
	}
	if p.SubmittedByUserID.Valid {
		item.SubmittedByUserId = uint32(p.SubmittedByUserID.Int32)
		if u := s.DB.GetUserByID(item.SubmittedByUserId); u != nil {
			item.SubmittedByUsername = u.Username
		}
	}
	if sa, err := s.DB.GetSocialAccount(p.SocialAccountID); err == nil && sa != nil {
		item.Post.SocialAccountIdentity = sa.Identity
		if s.cc != nil {
			if svc := s.cc.Get(sa.Connector); svc != nil {
				item.Post.SocialAccountIcon = svc.GetIcon()
			}
		}
	}
	if p.CampaignID.Valid {
		item.Post.CampaignId = uint32(p.CampaignID.Int32)
	}
	if p.CampaignName.Valid {
		item.Post.CampaignName = p.CampaignName.String
	}

	isSubmitter := p.SubmittedByUserID.Valid && uint32(p.SubmittedByUserID.Int32) == viewerID
	isAssignee := false
	if p.AccountPolicyID.Valid {
		if stage, err := s.DB.GetApprovalStageByPolicyAndOrder(uint32(p.AccountPolicyID.Int32), p.ApprovalStage); err == nil && stage != nil {
			ok, _ := s.DB.CanUserApproveStage(viewerID, stage)
			isAssignee = ok
			item.WaitingOn = describeStageAssignee(stage)
		}
	}

	item.CanApprove = isAssignee && !isSubmitter
	item.CanReject = isSubmitter || isAssignee
	return item
}

func describeStageAssignee(stage *db.AccountPolicyApprovalStage) string {
	if stage == nil {
		return ""
	}
	if stage.UserID.Valid {
		if stage.Username.Valid && stage.Username.String != "" {
			return stage.Username.String
		}
		return fmt.Sprintf("user #%d", stage.UserID.Int32)
	}
	if stage.UserGroupID.Valid {
		if stage.UserGroupName.Valid && stage.UserGroupName.String != "" {
			return "group: " + stage.UserGroupName.String
		}
		return fmt.Sprintf("group #%d", stage.UserGroupID.Int32)
	}
	return ""
}

func (s *ControlApi) ApprovePost(ctx context.Context, req *connect.Request[controlv1.ApprovePostRequest]) (*connect.Response[controlv1.ApprovePostResponse], error) {
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
	if post.State != db.PostStatePendingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post is not pending approval"))
	}
	if post.SubmittedByUserID.Valid && uint32(post.SubmittedByUserID.Int32) == au.User.ID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("submitter cannot approve their own post"))
	}
	if !post.AccountPolicyID.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post has no account policy"))
	}

	policyID := uint32(post.AccountPolicyID.Int32)
	stage, err := s.DB.GetApprovalStageByPolicyAndOrder(policyID, post.ApprovalStage)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("current approval stage not found"))
	}
	ok, err := s.DB.CanUserApproveStage(au.User.ID, stage)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not an approver for the current stage"))
	}

	if err := s.DB.RecordPostApproval(post.ID, stage.ID, au.User.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record approval"))
	}

	stageCount, err := s.DB.CountApprovalStages(policyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count stages"))
	}

	nextStage := post.ApprovalStage + 1
	if int(nextStage) < stageCount {
		if err := s.DB.AdvancePostApprovalStage(post.ID, nextStage); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to advance stage"))
		}
		post.ApprovalStage = nextStage
		return connect.NewResponse(&controlv1.ApprovePostResponse{
			StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Stage approved; more stages remain"},
			Post: &controlv1.PostStatus{
				Id:              post.ID,
				SocialAccountId: post.SocialAccountID,
				Content:         post.Content,
				State:           db.PostStatePendingApproval,
				Success:         true,
			},
		}), nil
	}

	socialAccount, err := s.DB.GetSocialAccount(post.SocialAccountID)
	if err != nil || socialAccount == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("social account not found"))
	}
	postStatus := s.releaseApprovedPost(ctx, post, socialAccount)
	log.Infof("Post %d fully approved and released (state=%s)", post.ID, postStatus.State)

	return connect.NewResponse(&controlv1.ApprovePostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post fully approved"},
		Post:             postStatus,
	}), nil
}

func (s *ControlApi) RejectPost(ctx context.Context, req *connect.Request[controlv1.RejectPostRequest]) (*connect.Response[controlv1.RejectPostResponse], error) {
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
	if post.State != db.PostStatePendingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post is not pending approval"))
	}

	isSubmitter := post.SubmittedByUserID.Valid && uint32(post.SubmittedByUserID.Int32) == au.User.ID
	canReject := isSubmitter
	if !canReject && post.AccountPolicyID.Valid {
		stage, stErr := s.DB.GetApprovalStageByPolicyAndOrder(uint32(post.AccountPolicyID.Int32), post.ApprovalStage)
		if stErr == nil {
			ok, _ := s.DB.CanUserApproveStage(au.User.ID, stage)
			canReject = ok
		}
	}
	if !canReject {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to reject this post"))
	}

	if err := s.DB.SetPostState(post.ID, db.PostStateRejected); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reject post"))
	}
	if req.Msg.Reason != "" {
		_ = s.DB.InsertTableLog(
			fmt.Sprintf("Post %d rejected by %s: %s", post.ID, au.User.Username, req.Msg.Reason),
			"info",
			&post.SocialAccountID,
		)
	}

	return connect.NewResponse(&controlv1.RejectPostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post rejected"},
	}), nil
}

func (s *ControlApi) marshalPostStatus(post *db.Post) *controlv1.PostStatus {
	if post == nil {
		return nil
	}
	ps := &controlv1.PostStatus{
		Id:              post.ID,
		SocialAccountId: post.SocialAccountID,
		Content:         post.Content,
		Success:         post.Status,
		PostUrl:         post.PostURL.String,
		State:           post.State,
		Created:         post.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	postedDate := post.CreatedAt
	if post.ScheduledAt.Valid {
		postedDate = post.ScheduledAt.Time
	}
	ps.PostedDate = postedDate.Format("2006-01-02 15:04:05")
	if post.CampaignID.Valid {
		ps.CampaignId = uint32(post.CampaignID.Int32)
	}
	if post.CampaignName.Valid {
		ps.CampaignName = post.CampaignName.String
	}
	if sa, err := s.DB.GetSocialAccount(post.SocialAccountID); err == nil && sa != nil {
		ps.SocialAccountIdentity = sa.Identity
		if s.cc != nil {
			if svc := s.cc.Get(sa.Connector); svc != nil {
				ps.SocialAccountIcon = svc.GetIcon()
			}
		}
	}
	return ps
}

func (s *ControlApi) GetPost(ctx context.Context, req *connect.Request[controlv1.GetPostRequest]) (*connect.Response[controlv1.GetPostResponse], error) {
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

	res := &controlv1.GetPostResponse{
		Post: s.marshalPostStatus(post),
	}

	if post.State == db.PostStatePendingApproval {
		pending := s.marshalPendingApproval(post, au.User.ID)
		res.ApprovalStage = pending.ApprovalStage
		res.AccountPolicyId = pending.AccountPolicyId
		res.AccountPolicyName = pending.AccountPolicyName
		res.SubmissionSource = pending.SubmissionSource
		res.SubmittedByUserId = pending.SubmittedByUserId
		res.SubmittedByUsername = pending.SubmittedByUsername
		res.CanApprove = pending.CanApprove
		res.CanReject = pending.CanReject
		res.CanEdit = pending.CanReject
		res.WaitingOn = pending.WaitingOn
	}

	return connect.NewResponse(res), nil
}

func (s *ControlApi) UpdatePendingPost(ctx context.Context, req *connect.Request[controlv1.UpdatePendingPostRequest]) (*connect.Response[controlv1.UpdatePendingPostResponse], error) {
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
	if post.State != db.PostStatePendingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post is not pending approval"))
	}

	isSubmitter := post.SubmittedByUserID.Valid && uint32(post.SubmittedByUserID.Int32) == au.User.ID
	canEdit := isSubmitter
	if !canEdit && post.AccountPolicyID.Valid {
		stage, stErr := s.DB.GetApprovalStageByPolicyAndOrder(uint32(post.AccountPolicyID.Int32), post.ApprovalStage)
		if stErr == nil {
			ok, _ := s.DB.CanUserApproveStage(au.User.ID, stage)
			canEdit = ok
		}
	}
	if !canEdit {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to edit this pending post"))
	}

	if err := s.DB.UpdatePostContent(post.ID, content); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update post content"))
	}

	post.Content = content
	log.Infof("Pending post %d content updated by %s", post.ID, au.User.Username)

	return connect.NewResponse(&controlv1.UpdatePendingPostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post content saved"},
		Post:             s.marshalPostStatus(post),
	}), nil
}
