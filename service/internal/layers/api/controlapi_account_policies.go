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
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/rbac"
	log "github.com/sirupsen/logrus"
)

func marshalPolicyStage(st *db.AccountPolicyApprovalStage) *controlv1.AccountPolicyApprovalStage {
	ps := &controlv1.AccountPolicyApprovalStage{
		Id:         st.ID,
		StageOrder: st.StageOrder,
	}
	if st.UserID.Valid {
		ps.UserId = intconv.ToUint32(st.UserID.Int32)
	}
	if st.UserGroupID.Valid {
		ps.UserGroupId = intconv.ToUint32(st.UserGroupID.Int32)
	}
	if st.Username.Valid {
		ps.Username = st.Username.String
	}
	if st.UserGroupName.Valid {
		ps.UserGroupName = st.UserGroupName.String
	}
	return ps
}

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
		out.Stages = append(out.Stages, marshalPolicyStage(st))
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
			dbStage.UserID = sql.NullInt32{Int32: intconv.ToInt32(st.UserId), Valid: true}
		} else {
			dbStage.UserGroupID = sql.NullInt32{Int32: intconv.ToInt32(st.UserGroupId), Valid: true}
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

func parsePolicyWriteInput(name string, stages []*controlv1.AccountPolicyApprovalStage) (string, []*db.AccountPolicyApprovalStage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	dbStages, err := validatePolicyStages(stages)
	if err != nil {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return name, dbStages, nil
}

func (s *ControlApi) attachPolicyRelations(id uint32, stages []*db.AccountPolicyApprovalStage, accountIDs []uint32, rollbackOnFail bool) error {
	if err := s.DB.SetAccountPolicyApprovalStages(id, stages); err != nil {
		if rollbackOnFail {
			_ = s.DB.DeleteAccountPolicy(id)
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save stages: %w", err))
	}
	if err := s.DB.SetAccountPolicySocialAccounts(id, accountIDs); err != nil {
		if rollbackOnFail {
			_ = s.DB.DeleteAccountPolicy(id)
		}
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to attach social accounts (already on another policy?)"))
	}
	return nil
}

func (s *ControlApi) CreateAccountPolicy(ctx context.Context, req *connect.Request[controlv1.CreateAccountPolicyRequest]) (*connect.Response[controlv1.CreateAccountPolicyResponse], error) {
	name, stages, err := parsePolicyWriteInput(req.Msg.Name, req.Msg.Stages)
	if err != nil {
		return nil, err
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
	if err := s.attachPolicyRelations(id, stages, req.Msg.SocialAccountIds, true); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.CreateAccountPolicyResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Policy created"},
		PolicyId:         id,
	}), nil
}

func (s *ControlApi) requireExistingPolicy(id uint32) error {
	if id == 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	if _, err := s.DB.GetAccountPolicy(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("policy not found"))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get policy"))
	}
	return nil
}

func (s *ControlApi) UpdateAccountPolicy(ctx context.Context, req *connect.Request[controlv1.UpdateAccountPolicyRequest]) (*connect.Response[controlv1.UpdateAccountPolicyResponse], error) {
	name, stages, err := parsePolicyWriteInput(req.Msg.Name, req.Msg.Stages)
	if err != nil {
		return nil, err
	}
	if err := s.requireExistingPolicy(req.Msg.Id); err != nil {
		return nil, err
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
	if err := s.attachPolicyRelations(req.Msg.Id, stages, req.Msg.SocialAccountIds, false); err != nil {
		return nil, err
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

func canViewAllPendingApprovals(au *authentication.AuthenticatedUser) bool {
	return au.HasPermission(rbac.PermissionAccountPoliciesManage) ||
		au.HasPermission(rbac.PermissionSocialAccountsViewAll) ||
		(au.RBAC != nil && au.RBAC.IsSuperuser)
}

func (s *ControlApi) ListPendingApprovals(ctx context.Context, req *connect.Request[controlv1.ListPendingApprovalsRequest]) (*connect.Response[controlv1.ListPendingApprovalsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	posts, err := s.DB.SelectPendingApprovalsVisibleToUser(au.User.ID, canViewAllPendingApprovals(au))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list pending approvals"))
	}

	out := make([]*controlv1.PendingApproval, 0, len(posts))
	for _, p := range posts {
		out = append(out, s.marshalPendingApproval(p, au.User.ID))
	}
	return connect.NewResponse(&controlv1.ListPendingApprovalsResponse{Pending: out}), nil
}

func pendingApprovalPostedDate(p *db.Post) string {
	postedDate := p.CreatedAt
	if p.ScheduledAt.Valid {
		postedDate = p.ScheduledAt.Time
	}
	return postedDate.Format("2006-01-02 15:04:05")
}

func (s *ControlApi) fillPendingApprovalPolicy(item *controlv1.PendingApproval, p *db.Post) {
	if !p.AccountPolicyID.Valid {
		return
	}
	item.AccountPolicyId = intconv.ToUint32(p.AccountPolicyID.Int32)
	if pol, err := s.DB.GetAccountPolicy(item.AccountPolicyId); err == nil && pol != nil {
		item.AccountPolicyName = pol.Name
	}
}

func (s *ControlApi) fillPendingApprovalSubmitter(item *controlv1.PendingApproval, p *db.Post) {
	if !p.SubmittedByUserID.Valid {
		return
	}
	item.SubmittedByUserId = intconv.ToUint32(p.SubmittedByUserID.Int32)
	if u := s.DB.GetUserByID(item.SubmittedByUserId); u != nil {
		item.SubmittedByUsername = u.Username
	}
}

func (s *ControlApi) fillPendingApprovalAccount(item *controlv1.PendingApproval, p *db.Post) {
	sa, err := s.DB.GetSocialAccount(p.SocialAccountIDUint())
	if err != nil || sa == nil {
		return
	}
	item.Post.SocialAccountIdentity = sa.Identity
	item.Post.SocialAccountIcon = s.connectorIcon(sa.Connector)
}

func fillPendingApprovalCampaign(item *controlv1.PendingApproval, p *db.Post) {
	if p.CampaignID.Valid {
		item.Post.CampaignId = intconv.ToUint32(p.CampaignID.Int32)
	}
	if p.CampaignName.Valid {
		item.Post.CampaignName = p.CampaignName.String
	}
}

func (s *ControlApi) pendingApprovalStageInfo(p *db.Post, viewerID uint32) (isAssignee bool, waitingOn string) {
	if !p.AccountPolicyID.Valid {
		return false, ""
	}
	stage, err := s.DB.GetApprovalStageByPolicyAndOrder(intconv.ToUint32(p.AccountPolicyID.Int32), p.ApprovalStage)
	if err != nil || stage == nil {
		return false, ""
	}
	ok, _ := s.DB.CanUserApproveStage(viewerID, stage)
	return ok, describeStageAssignee(stage)
}

func (s *ControlApi) pendingApprovalPermissions(p *db.Post, viewerID uint32) (canApprove, canReject bool, waitingOn string) {
	isSubmitter := p.SubmittedByUserID.Valid && intconv.ToUint32(p.SubmittedByUserID.Int32) == viewerID
	isAssignee, waitingOn := s.pendingApprovalStageInfo(p, viewerID)
	return isAssignee && !isSubmitter, isSubmitter || isAssignee, waitingOn
}

func (s *ControlApi) marshalPendingApproval(p *db.Post, viewerID uint32) *controlv1.PendingApproval {
	item := &controlv1.PendingApproval{
		Post: &controlv1.PostStatus{
			Id:              p.ID,
			SocialAccountId: p.SocialAccountIDUint(),
			Content:         p.Content,
			State:           p.State,
			Success:         false,
			PostUrl:         p.PostURL.String,
			Created:         p.CreatedAt.Format("2006-01-02 15:04:05"),
			PostedDate:      pendingApprovalPostedDate(p),
		},
		ApprovalStage:    p.ApprovalStage,
		SubmissionSource: p.SubmissionSource,
	}
	s.fillPendingApprovalPolicy(item, p)
	s.fillPendingApprovalSubmitter(item, p)
	s.fillPendingApprovalAccount(item, p)
	fillPendingApprovalCampaign(item, p)

	item.CanApprove, item.CanReject, item.WaitingOn = s.pendingApprovalPermissions(p, viewerID)
	return item
}

func describeUserAssignee(stage *db.AccountPolicyApprovalStage) string {
	if stage.Username.Valid && stage.Username.String != "" {
		return stage.Username.String
	}
	return fmt.Sprintf("user #%d", stage.UserID.Int32)
}

func describeGroupAssignee(stage *db.AccountPolicyApprovalStage) string {
	if stage.UserGroupName.Valid && stage.UserGroupName.String != "" {
		return "group: " + stage.UserGroupName.String
	}
	return fmt.Sprintf("group #%d", stage.UserGroupID.Int32)
}

func describeStageAssignee(stage *db.AccountPolicyApprovalStage) string {
	if stage == nil {
		return ""
	}
	if stage.UserID.Valid {
		return describeUserAssignee(stage)
	}
	if stage.UserGroupID.Valid {
		return describeGroupAssignee(stage)
	}
	return ""
}

func (s *ControlApi) loadPendingApprovalPost(postID uint32) (*db.Post, error) {
	if postID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id is required"))
	}
	post, err := s.DB.GetPost(postID)
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	if post.State != db.PostStatePendingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post is not pending approval"))
	}
	return post, nil
}

func rejectIfSubmitterApproving(au *authentication.AuthenticatedUser, post *db.Post) error {
	if post.SubmittedByUserID.Valid && intconv.ToUint32(post.SubmittedByUserID.Int32) == au.User.ID {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("submitter cannot approve their own post"))
	}
	return nil
}

func (s *ControlApi) loadCurrentApprovalStage(post *db.Post) (*db.AccountPolicyApprovalStage, error) {
	if !post.AccountPolicyID.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post has no account policy"))
	}
	stage, err := s.DB.GetApprovalStageByPolicyAndOrder(intconv.ToUint32(post.AccountPolicyID.Int32), post.ApprovalStage)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("current approval stage not found"))
	}
	return stage, nil
}

func (s *ControlApi) authorizeApprovePost(au *authentication.AuthenticatedUser, post *db.Post) (*db.AccountPolicyApprovalStage, error) {
	if err := rejectIfSubmitterApproving(au, post); err != nil {
		return nil, err
	}
	stage, err := s.loadCurrentApprovalStage(post)
	if err != nil {
		return nil, err
	}
	ok, err := s.DB.CanUserApproveStage(au.User.ID, stage)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not an approver for the current stage"))
	}
	return stage, nil
}

func (s *ControlApi) advanceApprovalStage(post *db.Post, nextStage uint32) (*controlv1.ApprovePostResponse, error) {
	if err := s.DB.AdvancePostApprovalStage(post.ID, nextStage); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to advance stage"))
	}
	post.ApprovalStage = nextStage
	return &controlv1.ApprovePostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Stage approved; more stages remain"},
		Post: &controlv1.PostStatus{
			Id: post.ID, SocialAccountId: post.SocialAccountIDUint(), Content: post.Content,
			State: db.PostStatePendingApproval, Success: true,
		},
	}, nil
}

func (s *ControlApi) releaseFullyApprovedPost(ctx context.Context, post *db.Post) (*controlv1.ApprovePostResponse, error) {
	socialAccount, err := s.DB.GetSocialAccount(post.SocialAccountIDUint())
	if err != nil || socialAccount == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("social account not found"))
	}
	postStatus := s.releaseApprovedPost(ctx, post, socialAccount)
	log.Infof("Post %d fully approved and released (state=%s)", post.ID, postStatus.State)
	return &controlv1.ApprovePostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post fully approved"},
		Post:             postStatus,
	}, nil
}

func (s *ControlApi) advanceOrReleaseApprovedPost(ctx context.Context, post *db.Post, policyID uint32) (*controlv1.ApprovePostResponse, error) {
	stageCount, err := s.DB.CountApprovalStages(policyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count stages"))
	}
	nextStage := post.ApprovalStage + 1
	if int(nextStage) < stageCount {
		return s.advanceApprovalStage(post, nextStage)
	}
	return s.releaseFullyApprovedPost(ctx, post)
}

func (s *ControlApi) recordAndAdvanceApproval(ctx context.Context, au *authentication.AuthenticatedUser, post *db.Post, stage *db.AccountPolicyApprovalStage) (*controlv1.ApprovePostResponse, error) {
	if err := s.DB.RecordPostApproval(post.ID, stage.ID, au.User.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record approval"))
	}
	return s.advanceOrReleaseApprovedPost(ctx, post, intconv.ToUint32(post.AccountPolicyID.Int32))
}

func (s *ControlApi) ApprovePost(ctx context.Context, req *connect.Request[controlv1.ApprovePostRequest]) (*connect.Response[controlv1.ApprovePostResponse], error) {
	au, post, stage, err := s.prepareApprovePost(ctx, req.Msg.PostId)
	if err != nil {
		return nil, err
	}
	resp, err := s.recordAndAdvanceApproval(ctx, au, post, stage)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *ControlApi) prepareApprovePost(ctx context.Context, postID uint32) (*authentication.AuthenticatedUser, *db.Post, *db.AccountPolicyApprovalStage, error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	post, err := s.loadPendingApprovalPost(postID)
	if err != nil {
		return nil, nil, nil, err
	}
	stage, err := s.authorizeApprovePost(au, post)
	if err != nil {
		return nil, nil, nil, err
	}
	return au, post, stage, nil
}

func (s *ControlApi) userCanRejectOrEditPending(auUserID uint32, post *db.Post) bool {
	if post.SubmittedByUserID.Valid && intconv.ToUint32(post.SubmittedByUserID.Int32) == auUserID {
		return true
	}
	if !post.AccountPolicyID.Valid {
		return false
	}
	stage, stErr := s.DB.GetApprovalStageByPolicyAndOrder(intconv.ToUint32(post.AccountPolicyID.Int32), post.ApprovalStage)
	if stErr != nil {
		return false
	}
	ok, _ := s.DB.CanUserApproveStage(auUserID, stage)
	return ok
}

func (s *ControlApi) logPostRejection(au *authentication.AuthenticatedUser, post *db.Post, reason string) {
	if reason == "" {
		return
	}
	s.DB.InsertTableLog(
		fmt.Sprintf("Post %d rejected by %s: %s", post.ID, au.User.Username, reason),
		"info",
		post.SocialAccountIDPtr(),
	)
}

func (s *ControlApi) RejectPost(ctx context.Context, req *connect.Request[controlv1.RejectPostRequest]) (*connect.Response[controlv1.RejectPostResponse], error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}
	post, err := s.loadPendingApprovalPost(req.Msg.PostId)
	if err != nil {
		return nil, err
	}
	if !s.userCanRejectOrEditPending(au.User.ID, post) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to reject this post"))
	}
	if err := s.DB.SetPostState(post.ID, db.PostStateRejected); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reject post"))
	}
	s.logPostRejection(au, post, req.Msg.Reason)
	return connect.NewResponse(&controlv1.RejectPostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post rejected"},
	}), nil
}

func fillPostStatusCampaign(ps *controlv1.PostStatus, post *db.Post) {
	if post.CampaignID.Valid {
		ps.CampaignId = intconv.ToUint32(post.CampaignID.Int32)
	}
	if post.CampaignName.Valid {
		ps.CampaignName = post.CampaignName.String
	}
}

func (s *ControlApi) fillPostStatusAccount(ps *controlv1.PostStatus, post *db.Post) {
	sa, err := s.DB.GetSocialAccount(post.SocialAccountIDUint())
	if err == nil && sa != nil {
		ps.SocialAccountIdentity = sa.Identity
		ps.SocialAccountIcon = s.connectorIcon(sa.Connector)
		return
	}
	if post.State == db.PostStateDraft {
		ps.SocialAccountIdentity = "Draft"
		ps.SocialAccountIcon = "mdi:file-document-outline"
		return
	}
	ps.SocialAccountIdentity = "Unknown"
	ps.SocialAccountIcon = "mdi:question-mark-circle"
}

func (s *ControlApi) marshalPostStatus(post *db.Post) *controlv1.PostStatus {
	if post == nil {
		return nil
	}
	ps := &controlv1.PostStatus{
		Id:              post.ID,
		SocialAccountId: post.SocialAccountIDUint(),
		Content:         post.Content,
		Success:         post.Status,
		PostUrl:         post.PostURL.String,
		State:           post.State,
		Created:         post.CreatedAt.Format("2006-01-02 15:04:05"),
		PostedDate:      pendingApprovalPostedDate(post),
	}
	fillPostStatusCampaign(ps, post)
	s.fillPostStatusAccount(ps, post)
	return ps
}

func fillGetPostPendingFields(res *controlv1.GetPostResponse, pending *controlv1.PendingApproval) {
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

func (s *ControlApi) fillGetPostCapabilities(res *controlv1.GetPostResponse, post *db.Post, viewerID uint32) {
	switch post.State {
	case db.PostStatePendingApproval:
		fillGetPostPendingFields(res, s.marshalPendingApproval(post, viewerID))
	case db.PostStateDraft:
		if post.SubmittedByUserID.Valid && intconv.ToUint32(post.SubmittedByUserID.Int32) == viewerID {
			res.CanEdit = true
		}
	}
}

func requirePostID(postID uint32) error {
	if postID == 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id is required"))
	}
	return nil
}

func (s *ControlApi) loadPostByID(postID uint32) (*db.Post, error) {
	post, err := s.DB.GetPost(postID)
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	return post, nil
}

func (s *ControlApi) GetPost(ctx context.Context, req *connect.Request[controlv1.GetPostRequest]) (*connect.Response[controlv1.GetPostResponse], error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}
	if idErr := requirePostID(req.Msg.PostId); idErr != nil {
		return nil, idErr
	}
	post, err := s.loadPostByID(req.Msg.PostId)
	if err != nil {
		return nil, err
	}
	res := &controlv1.GetPostResponse{Post: s.marshalPostStatus(post)}
	s.fillGetPostCapabilities(res, post, au.User.ID)
	return connect.NewResponse(res), nil
}

func requireNonEmptyPendingContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is required"))
	}
	return content, nil
}

func (s *ControlApi) savePendingPostContent(au *authentication.AuthenticatedUser, post *db.Post, content string) error {
	if !s.userCanRejectOrEditPending(au.User.ID, post) {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to edit this pending post"))
	}
	if err := s.DB.UpdatePostContent(post.ID, content); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update post content"))
	}
	post.Content = content
	log.Infof("Pending post %d content updated by %s", post.ID, au.User.Username)
	return nil
}

func (s *ControlApi) UpdatePendingPost(ctx context.Context, req *connect.Request[controlv1.UpdatePendingPostRequest]) (*connect.Response[controlv1.UpdatePendingPostResponse], error) {
	au, post, content, err := s.prepareUpdatePendingPost(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	if err := s.savePendingPostContent(au, post, content); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.UpdatePendingPostResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Post content saved"},
		Post:             s.marshalPostStatus(post),
	}), nil
}

func (s *ControlApi) prepareUpdatePendingPost(ctx context.Context, msg *controlv1.UpdatePendingPostRequest) (*authentication.AuthenticatedUser, *db.Post, string, error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	content, err := requireNonEmptyPendingContent(msg.Content)
	if err != nil {
		return nil, nil, "", err
	}
	post, err := s.loadPendingApprovalPost(msg.PostId)
	if err != nil {
		return nil, nil, "", err
	}
	if !s.userCanRejectOrEditPending(au.User.ID, post) {
		return nil, nil, "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to edit this pending post"))
	}
	return au, post, content, nil
}
