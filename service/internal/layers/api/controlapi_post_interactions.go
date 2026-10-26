package api

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/postreplies"
	"github.com/jamesread/japella/internal/rbac"
)

// userCanPostWithSocialAccount reports whether the user may act as the account
// (post, like). Mirrors userCanReadSocialAccount with "post" access.
func (s *ControlApi) userCanPostWithSocialAccount(au *authentication.AuthenticatedUser, socialAccountID uint32) bool {
	if au == nil || au.User == nil || socialAccountID == 0 {
		return false
	}
	if au.RBAC != nil && au.RBAC.IsSuperuser {
		return true
	}
	if au.HasPermission(rbac.PermissionSocialAccountsViewAll) {
		return true
	}
	sa, err := s.DB.GetSocialAccount(socialAccountID)
	if err != nil || sa == nil {
		return false
	}
	if sa.OwnerUserID.Valid && uint32(sa.OwnerUserID.Int32) == au.User.ID {
		return true
	}
	return s.DB.CanUserAccessSocialAccount(au.User.ID, socialAccountID, "post")
}

func (s *ControlApi) connectorSupportsLikes(connectorName string) bool {
	if s.cc == nil {
		return false
	}
	_, ok := s.cc.Get(connectorName).(connector.ConnectorWithPostLikes)
	return ok
}

// interactionTarget picks the id to act on: a boost's original when known.
func interactionTarget(ref *controlv1.PostRef) (remoteID, remoteURL string) {
	remoteID = ref.GetInteractionRemoteId()
	if remoteID == "" {
		remoteID = ref.GetRemoteId()
	}
	return remoteID, ref.GetRemoteUrl()
}

// postRefAccount validates a PostRef and loads its social account.
func (s *ControlApi) postRefAccount(ref *controlv1.PostRef) (*db.SocialAccount, error) {
	if ref == nil || ref.GetSocialAccountId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post.social_account_id is required"))
	}
	if ref.GetRemoteId() == "" && ref.GetRemoteUrl() == "" && ref.GetInteractionRemoteId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post remote id or url is required"))
	}
	sa, err := s.DB.GetSocialAccount(ref.GetSocialAccountId())
	if err != nil || sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	return sa, nil
}

// freshSocialAccount refreshes an OAuth token that is about to expire.
func (s *ControlApi) freshSocialAccount(sa *db.SocialAccount, svc connector.BaseConnector) *db.SocialAccount {
	if socialAccountNeedsTokenRefresh(sa, time.Hour) {
		if refreshed, err := s.refreshSocialAccountToken(sa, svc); err == nil {
			return refreshed
		}
	}
	return sa
}

func (s *ControlApi) SetPostLiked(ctx context.Context, req *connect.Request[controlv1.SetPostLikedRequest]) (*connect.Response[controlv1.SetPostLikedResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	ref := req.Msg.GetPost()
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanPostWithSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("post access to this social account is required to like as it"))
	}

	svc := s.cc.Get(sa.Connector)
	lc, ok := svc.(connector.ConnectorWithPostLikes)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("liking isn't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	result, err := lc.SetPostLiked(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL, req.Msg.GetLiked())
	if rl := connector.AsRateLimited(err); rl != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited by %s; try again in %s", sa.Connector, rl.RetryAfter))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to update like: %w", err))
	}

	if id := ref.GetFeedPostId(); id != 0 {
		if entry, getErr := s.DB.GetFeedEntryByID(id); getErr == nil && entry != nil && entry.SocialAccountID == sa.ID {
			_ = s.DB.SetFeedEntryLiked(id, result.Liked, result.LikeCount)
		}
	}

	return connect.NewResponse(&controlv1.SetPostLikedResponse{Liked: result.Liked, LikeCount: result.LikeCount}), nil
}

func (s *ControlApi) GetPostThread(ctx context.Context, req *connect.Request[controlv1.GetPostThreadRequest]) (*connect.Response[controlv1.GetPostThreadResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	ref := req.Msg.GetPost()
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanReadSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}

	svc := s.cc.Get(sa.Connector)
	rc, ok := svc.(connector.ConnectorWithPostReplies)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("comment threads aren't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	fetched, err := rc.FetchPostReplies(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL)
	if rl := connector.AsRateLimited(err); rl != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited by %s; try again in %s", sa.Connector, rl.RetryAfter))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to fetch comments: %w", err))
	}

	rows := postreplies.ToDB(0, fetched)

	// If the account already replied (here or on the network directly),
	// remember it so the feed shows the post as commented.
	commented := false
	for _, r := range rows {
		if r.AuthorHandle != "" || r.AuthorName != "" {
			if feedauthor.IsOwnPost(sa.Identity, r.AuthorHandle, r.AuthorName) {
				commented = true
				break
			}
		}
	}
	if commented {
		s.recordPostCommented(sa.ID, ref)
	}

	return connect.NewResponse(&controlv1.GetPostThreadResponse{
		Replies:    replyNodesToProto(postreplies.BuildThread(rows)),
		ReplyCount: uint32(len(rows)),
		Commented:  commented,
	}), nil
}

// postCommentMaxRunes is a generous ceiling; networks enforce their own
// (shorter) limits and their error is passed back.
const postCommentMaxRunes = 5000

func (s *ControlApi) connectorSupportsReplying(connectorName string) bool {
	if s.cc == nil {
		return false
	}
	_, ok := s.cc.Get(connectorName).(connector.ConnectorWithPostReplying)
	return ok
}

func (s *ControlApi) recordPostCommented(socialAccountID uint32, ref *controlv1.PostRef) {
	if key := feedauthor.PostKey(ref.GetRemoteUrl(), ref.GetRemoteId()); key != "" {
		_ = s.DB.RecordPostCommented(socialAccountID, feedauthor.KeyHash(key), key)
	}
}

// PostComment publishes a reply to a feed or feed column post as its social
// account. Comments are published immediately, so accounts whose approval
// policy would hold a post can't comment from here.
func (s *ControlApi) PostComment(ctx context.Context, req *connect.Request[controlv1.PostCommentRequest]) (*connect.Response[controlv1.PostCommentResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	ref := req.Msg.GetPost()
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanPostWithSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("post access to this social account is required to comment as it"))
	}
	if s.shouldHoldForApproval(sa.ID, SubmissionSourceFromContext(ctx)) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("posts from %s need approval, so comments can't be sent from the feed", sa.Identity))
	}

	content := strings.TrimSpace(req.Msg.GetContent())
	if content == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("comment is empty"))
	}
	if utf8.RuneCountInString(content) > postCommentMaxRunes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("comment must be at most %d characters", postCommentMaxRunes))
	}

	svc := s.cc.Get(sa.Connector)
	rc, ok := svc.(connector.ConnectorWithPostReplying)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("commenting isn't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	reply, err := rc.ReplyToPost(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL, content)
	if rl := connector.AsRateLimited(err); rl != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited by %s; try again in %s", sa.Connector, rl.RetryAfter))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to post comment: %w", err))
	}

	s.recordPostCommented(sa.ID, ref)

	out := &controlv1.PostCommentResponse{}
	if reply != nil {
		rows := postreplies.ToDB(0, &connector.PostReplies{Replies: []*connector.PostReply{reply}})
		if converted := replyNodesToProto(postreplies.BuildThread(rows)); len(converted) == 1 {
			out.Reply = converted[0]
		}
	}
	return connect.NewResponse(out), nil
}
