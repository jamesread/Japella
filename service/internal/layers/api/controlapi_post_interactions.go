package api

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/postreplies"
)

// userCanPostWithSocialAccount reports whether the user may act as the account
// (post, like). Mirrors userCanReadSocialAccount with "post" access.
func authenticatedPoster(au *authentication.AuthenticatedUser, socialAccountID uint32) bool {
	return au != nil && au.User != nil && socialAccountID != 0
}

func (s *ControlApi) userCanPostWithSocialAccount(au *authentication.AuthenticatedUser, socialAccountID uint32) bool {
	if !authenticatedPoster(au, socialAccountID) {
		return false
	}
	if canViewAllSocialAccounts(au) {
		return true
	}
	return s.userOwnsOrHasPostAccess(au, socialAccountID)
}

func (s *ControlApi) userOwnsOrHasPostAccess(au *authentication.AuthenticatedUser, socialAccountID uint32) bool {
	sa, err := s.DB.GetSocialAccount(socialAccountID)
	if err != nil || sa == nil {
		return false
	}
	if sa.OwnerUserID.Valid && intconv.ToUint32(sa.OwnerUserID.Int32) == au.User.ID {
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
	if err := validatePostRef(ref); err != nil {
		return nil, err
	}
	sa, err := s.DB.GetSocialAccount(ref.GetSocialAccountId())
	if err != nil || sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	return sa, nil
}

func postRefHasRemote(ref *controlv1.PostRef) bool {
	return ref.GetRemoteId() != "" || ref.GetRemoteUrl() != "" || ref.GetInteractionRemoteId() != ""
}

func validatePostRef(ref *controlv1.PostRef) error {
	if ref == nil || ref.GetSocialAccountId() == 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post.social_account_id is required"))
	}
	if !postRefHasRemote(ref) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post remote id or url is required"))
	}
	return nil
}

// freshSocialAccount refreshes an OAuth token that is about to expire.
func (s *ControlApi) freshSocialAccount(sa *db.SocialAccount, svc connector.BaseConnector) *db.SocialAccount {
	if socialAccountNeedsTokenRefresh(sa) {
		if refreshed, err := s.refreshSocialAccountToken(sa, svc); err == nil {
			return refreshed
		}
	}
	return sa
}

func (s *ControlApi) authorizePostInteraction(ctx context.Context, ref *controlv1.PostRef) (*authentication.AuthenticatedUser, *db.SocialAccount, error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, nil, err
	}
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, nil, err
	}
	if !s.userCanPostWithSocialAccount(au, sa.ID) {
		return nil, nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("post access to this social account is required to like as it"))
	}
	return au, sa, nil
}

func (s *ControlApi) SetPostLiked(ctx context.Context, req *connect.Request[controlv1.SetPostLikedRequest]) (*connect.Response[controlv1.SetPostLikedResponse], error) {
	ref := req.Msg.GetPost()
	_, sa, err := s.authorizePostInteraction(ctx, ref)
	if err != nil {
		return nil, err
	}

	svc := s.cc.Get(sa.Connector)
	lc, ok := svc.(connector.ConnectorWithPostLikes)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("liking isn't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	result, err := lc.SetPostLiked(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL, req.Msg.GetLiked())
	if err != nil {
		return nil, rateLimitedOrUnavailable(sa.Connector, err, "failed to update like")
	}

	s.persistFeedEntryLiked(ref, sa.ID, result)
	s.recordLikeActivity(sa, ref, result.Liked)

	return connect.NewResponse(&controlv1.SetPostLikedResponse{Liked: result.Liked, LikeCount: result.LikeCount}), nil
}

func (s *ControlApi) persistFeedEntryLiked(ref *controlv1.PostRef, socialAccountID uint32, result *connector.LikeResult) {
	id := ref.GetFeedPostId()
	if id == 0 {
		return
	}
	entry, getErr := s.DB.GetFeedEntryByID(id)
	if getErr == nil && entry != nil && entry.SocialAccountID == socialAccountID {
		_ = s.DB.SetFeedEntryLiked(id, result.Liked, result.LikeCount)
	}
}

func (s *ControlApi) authorizeReadPostRef(ctx context.Context, ref *controlv1.PostRef) (*db.SocialAccount, error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanReadSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}
	return sa, nil
}

func (s *ControlApi) GetPostThread(ctx context.Context, req *connect.Request[controlv1.GetPostThreadRequest]) (*connect.Response[controlv1.GetPostThreadResponse], error) {
	ref := req.Msg.GetPost()
	sa, err := s.authorizeReadPostRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	svc := s.cc.Get(sa.Connector)
	rc, ok := svc.(connector.ConnectorWithPostReplies)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("comment threads aren't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	fetched, err := rc.FetchPostReplies(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL)
	if err != nil {
		return nil, rateLimitedOrUnavailable(sa.Connector, err, "failed to fetch comments")
	}

	rows := postreplies.ToDB(0, fetched)
	commented := threadHasOwnReply(sa.Identity, rows)
	if commented {
		s.recordPostCommented(sa.ID, ref)
	}

	return connect.NewResponse(&controlv1.GetPostThreadResponse{
		Replies:    replyNodesToProto(postreplies.BuildThread(rows)),
		ReplyCount: intconv.ToUint32(len(rows)),
		Commented:  commented,
	}), nil
}

func threadHasOwnReply(identity string, rows []*db.PostReply) bool {
	for _, r := range rows {
		if r.AuthorHandle == "" && r.AuthorName == "" {
			continue
		}
		if feedauthor.IsOwnPost(identity, r.AuthorHandle, r.AuthorName) {
			return true
		}
	}
	return false
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

func validateCommentContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("comment is empty"))
	}
	if utf8.RuneCountInString(content) > postCommentMaxRunes {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("comment must be at most %d characters", postCommentMaxRunes))
	}
	return content, nil
}

// PostComment publishes a reply to a feed or feed column post as its social
// account. Comments are published immediately, so accounts whose approval
// policy would hold a post can't comment from here.
func (s *ControlApi) PostComment(ctx context.Context, req *connect.Request[controlv1.PostCommentRequest]) (*connect.Response[controlv1.PostCommentResponse], error) {
	ref := req.Msg.GetPost()
	_, sa, err := s.authorizeComment(ctx, ref)
	if err != nil {
		return nil, err
	}

	content, err := validateCommentContent(req.Msg.GetContent())
	if err != nil {
		return nil, err
	}

	svc := s.cc.Get(sa.Connector)
	rc, ok := svc.(connector.ConnectorWithPostReplying)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("commenting isn't available for %s", sa.Connector))
	}

	remoteID, remoteURL := interactionTarget(ref)
	reply, err := rc.ReplyToPost(toConnectorSA(s.freshSocialAccount(sa, svc)), remoteID, remoteURL, content)
	if err != nil {
		return nil, rateLimitedOrUnavailable(sa.Connector, err, "failed to post comment")
	}

	s.recordPostCommented(sa.ID, ref)
	s.recordCommentActivity(sa, ref, content, reply)

	return connect.NewResponse(commentReplyResponse(reply)), nil
}

func (s *ControlApi) authorizeComment(ctx context.Context, ref *controlv1.PostRef) (*authentication.AuthenticatedUser, *db.SocialAccount, error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, nil, err
	}
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, nil, err
	}
	if !s.userCanPostWithSocialAccount(au, sa.ID) {
		return nil, nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("post access to this social account is required to comment as it"))
	}
	if s.shouldHoldForApproval(sa.ID, SubmissionSourceFromContext(ctx)) {
		return nil, nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("posts from %s need approval, so comments can't be sent from the feed", sa.Identity))
	}
	return au, sa, nil
}

func commentReplyResponse(reply *connector.PostReply) *controlv1.PostCommentResponse {
	out := &controlv1.PostCommentResponse{}
	if reply == nil {
		return out
	}
	rows := postreplies.ToDB(0, &connector.PostReplies{Replies: []*connector.PostReply{reply}})
	if converted := replyNodesToProto(postreplies.BuildThread(rows)); len(converted) == 1 {
		out.Reply = converted[0]
	}
	return out
}
