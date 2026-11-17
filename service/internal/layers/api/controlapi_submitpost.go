package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/layers/authentication"
	log "github.com/sirupsen/logrus"
)

// postSubmission holds what every post created by one SubmitPost call shares.
type postSubmission struct {
	msg         *controlv1.SubmitPostRequest
	source      string
	submittedBy sql.NullInt32
	scheduledAt *time.Time
}

func (s *ControlApi) SubmitPost(ctx context.Context, req *connect.Request[controlv1.SubmitPostRequest]) (*connect.Response[controlv1.SubmitPostResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	sub := &postSubmission{
		msg:         req.Msg,
		source:      SubmissionSourceFromContext(ctx),
		submittedBy: submittedByUser(au),
		scheduledAt: parseScheduledAt(req.Msg.ScheduledAt),
	}

	log.Infof("Received post request for social accounts: %+v (source=%s)", req.Msg.SocialAccounts, sub.source)

	if err := s.checkSourceDraft(au, req.Msg.FromDraftPostId); err != nil {
		return nil, err
	}

	if isUnassignedDraft(req.Msg) {
		return s.saveUnassignedDraft(sub)
	}

	posts, err := s.submitToAccounts(ctx, au, sub)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.SubmitPostResponse{Posts: posts}), nil
}

// checkSourceDraft verifies that a draft named by from_draft_post_id exists
// and belongs to the caller.
func (s *ControlApi) checkSourceDraft(au *authentication.AuthenticatedUser, draftID uint32) error {
	if draftID == 0 {
		return nil
	}
	userID, ok := authenticatedUserID(au)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	draft := s.loadDraft(draftID)
	if draft == nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from_draft_post_id is invalid"))
	}
	if !isSubmittedBy(draft, userID) {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not allowed to use this draft"))
	}
	return nil
}

// loadDraft returns the post if it exists and is still a draft.
func (s *ControlApi) loadDraft(postID uint32) *db.Post {
	post, err := s.DB.GetPost(postID)
	if err != nil || post == nil || post.State != db.PostStateDraft {
		return nil
	}
	return post
}

func isSubmittedBy(post *db.Post, userID uint32) bool {
	return post.SubmittedByUserID.Valid && intconv.ToUint32(post.SubmittedByUserID.Int32) == userID
}

func authenticatedUserID(au *authentication.AuthenticatedUser) (uint32, bool) {
	if au == nil || au.User == nil {
		return 0, false
	}
	return au.User.ID, true
}

func submittedByUser(au *authentication.AuthenticatedUser) sql.NullInt32 {
	userID, ok := authenticatedUserID(au)
	if !ok {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: intconv.ToInt32(userID), Valid: true}
}

// parseScheduledAt accepts RFC3339 or the "YYYY-MM-DDTHH:MM" a datetime-local
// input sends. An unparseable value posts immediately, as before.
func parseScheduledAt(value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}
	log.Warnf("Invalid scheduled_at format: %s", value)
	return nil
}

func isUnassignedDraft(msg *controlv1.SubmitPostRequest) bool {
	return msg.SaveAsDraft && len(msg.SocialAccounts) == 0
}

func (s *ControlApi) saveUnassignedDraft(sub *postSubmission) (*connect.Response[controlv1.SubmitPostResponse], error) {
	if len(sub.msg.Content) == 0 && len(sub.msg.MediaUrls) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("draft requires content or media"))
	}
	status := &controlv1.PostStatus{Content: sub.msg.Content}
	s.recordPost(sub, sub.newPost(db.PostStateDraft), status)
	return connect.NewResponse(&controlv1.SubmitPostResponse{Posts: []*controlv1.PostStatus{status}}), nil
}

func (s *ControlApi) submitToAccounts(ctx context.Context, au *authentication.AuthenticatedUser, sub *postSubmission) ([]*controlv1.PostStatus, error) {
	canViewAll := canViewAllSocialAccounts(au)
	posts := make([]*controlv1.PostStatus, 0, len(sub.msg.SocialAccounts))
	for _, accountID := range sub.msg.SocialAccounts {
		log.Infof("Processing post for account: %v", accountID)
		if !s.canPostToAccount(au, canViewAll, accountID) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("no post access to social account %d", accountID))
		}
		posts = append(posts, s.submitToAccount(ctx, sub, accountID))
	}
	return posts, nil
}

func (s *ControlApi) canPostToAccount(au *authentication.AuthenticatedUser, canViewAll bool, accountID uint32) bool {
	return au == nil || canViewAll || s.DB.CanUserAccessSocialAccount(au.User.ID, accountID, "post")
}

func (s *ControlApi) submitToAccount(ctx context.Context, sub *postSubmission, accountID uint32) *controlv1.PostStatus {
	status := &controlv1.PostStatus{SocialAccountId: accountID, Content: sub.msg.Content}

	sa, _ := s.DB.GetSocialAccount(accountID)
	if sa == nil {
		status.State = db.PostStateError
		return status
	}

	switch {
	case sub.msg.SaveAsDraft:
		s.saveAccountDraft(sub, sa, status)
	case s.shouldHoldForApproval(sa.ID, sub.source):
		s.holdForApproval(ctx, sub, sa, status)
	case sub.isScheduledLater():
		s.schedulePost(sub, sa, status)
	default:
		s.postNow(ctx, sub, sa, status)
	}
	return status
}

func (s *ControlApi) saveAccountDraft(sub *postSubmission, sa *db.SocialAccount, status *controlv1.PostStatus) {
	if s.recordPost(sub, sub.newAccountPost(sa, db.PostStateDraft), status) {
		status.SocialAccountIdentity = sa.Identity
	}
}

func (s *ControlApi) holdForApproval(ctx context.Context, sub *postSubmission, sa *db.SocialAccount, status *controlv1.PostStatus) {
	policy, _ := s.DB.GetAccountPolicyForSocialAccount(sa.ID)
	post := sub.newAccountPost(sa, db.PostStatePendingApproval)
	post.ScheduledAt = nullTime(sub.scheduledAt)
	post.AccountPolicyID = sql.NullInt32{Int32: intconv.ToInt32(policy.ID), Valid: true}
	if s.recordPost(sub, post, status) {
		status.SocialAccountIdentity = sa.Identity
		s.dispatchApprovalRequested(ctx, post, sa, policy)
	}
}

func (s *ControlApi) schedulePost(sub *postSubmission, sa *db.SocialAccount, status *controlv1.PostStatus) {
	post := sub.newAccountPost(sa, db.PostStateScheduled)
	post.ScheduledAt = nullTime(sub.scheduledAt)
	s.recordPost(sub, post, status)
}

// postNow publishes first, then records the outcome. A post that was published
// but could not be recorded still reports the publish result.
func (s *ControlApi) postNow(ctx context.Context, sub *postSubmission, sa *db.SocialAccount, status *controlv1.PostStatus) {
	s.tryPostStatus(sub.msg.Content, sub.msg.MediaUrls, sa, status)

	state := db.PostStateCompleted
	if !status.Success {
		state = db.PostStateError
	}

	post := sub.newAccountPost(sa, state)
	post.Status = status.Success
	post.PostURL = sql.NullString{String: status.PostUrl, Valid: status.PostUrl != ""}
	if err := s.DB.CreatePost(post); err != nil {
		log.Errorf("Failed to record post to account %d: %v", sa.ID, err)
	}
	s.savePostMedia(post.ID, sub.msg.MediaUrls)
	status.Id = post.ID
	status.State = state
	s.copyLlmToolRunsFromDraft(sub.msg.FromDraftPostId, post.ID)
	s.dispatchPostOutcome(ctx, post, sa, status.Success, status.PostUrl)
}

// recordPost saves a post that is not published yet and fills in status.
// It reports whether the post was saved.
func (s *ControlApi) recordPost(sub *postSubmission, post *db.Post, status *controlv1.PostStatus) bool {
	if err := s.DB.CreatePost(post); err != nil {
		log.Errorf("Failed to create %s post: %v", post.State, err)
		status.State = db.PostStateError
		return false
	}
	s.savePostMedia(post.ID, sub.msg.MediaUrls)
	status.Id = post.ID
	status.Success = true
	status.State = post.State
	s.copyLlmToolRunsFromDraft(sub.msg.FromDraftPostId, post.ID)
	return true
}

func (sub *postSubmission) newPost(state string) *db.Post {
	return &db.Post{
		Content:           sub.msg.Content,
		State:             state,
		CampaignID:        nullCampaignID(sub.msg.CampaignId),
		SubmissionSource:  sub.source,
		SubmittedByUserID: sub.submittedBy,
	}
}

func (sub *postSubmission) newAccountPost(sa *db.SocialAccount, state string) *db.Post {
	post := sub.newPost(state)
	post.SocialAccountID = db.NullSocialAccountID(sa.ID)
	return post
}

// Scheduled times within a few seconds of now are posted immediately.
func (sub *postSubmission) isScheduledLater() bool {
	return sub.scheduledAt != nil && sub.scheduledAt.After(time.Now().Add(5*time.Second))
}

func nullCampaignID(campaignID uint32) sql.NullInt32 {
	if campaignID == 0 {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: intconv.ToInt32(campaignID), Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
