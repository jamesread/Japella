package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/utils"
)

func (s *ControlApi) connectorSupportsRefetch(connectorName string) bool {
	if s.cc == nil {
		return false
	}
	_, ok := s.cc.Get(connectorName).(connector.ConnectorWithFeedPostRefetch)
	return ok
}

// fetchedPostToProto renders a reloaded feed column post. It isn't stored and
// comes from strangers, so like column search results its content is reduced
// to escaped text.
func (s *ControlApi) fetchedPostToProto(p *connector.FeedPost, sa *db.SocialAccount) *controlv1.FeedPost {
	icon := "mdi:question-mark-circle"
	if svc := s.cc.Get(sa.Connector); svc != nil {
		icon = svc.GetIcon()
	}
	out := &controlv1.FeedPost{
		SocialAccountId:          sa.ID,
		SocialAccountIcon:        icon,
		SocialAccountIdentity:    sa.Identity,
		SocialAccountConnector:   sa.Connector,
		Content:                  utils.TextToSafeHTML(utils.HTMLToText(p.Content)),
		AuthorId:                 p.AuthorID,
		AuthorName:               p.AuthorName,
		AuthorAvatarUrl:          safeSnapshotURL(p.AuthorAvatarURL),
		AuthorHandle:             p.AuthorHandle,
		AuthorProfileUrl:         safeSnapshotURL(p.AuthorProfileURL),
		OriginalAuthorId:         p.OriginalAuthorID,
		OriginalAuthorName:       p.OriginalAuthorName,
		OriginalAuthorAvatarUrl:  safeSnapshotURL(p.OriginalAuthorAvatarURL),
		OriginalAuthorHandle:     p.OriginalAuthorHandle,
		OriginalAuthorProfileUrl: safeSnapshotURL(p.OriginalAuthorProfileURL),
		RemoteUrl:                p.RemoteURL,
		RemoteId:                 p.RemoteID,
		PreviewUrl:               safeSnapshotURL(p.PreviewURL),
		PreviewTitle:             p.PreviewTitle,
		PreviewDescription:       utils.HTMLToText(p.PreviewDescription),
		PreviewImageUrl:          safeSnapshotURL(p.PreviewImageURL),
		Language:                 p.Language,

		FromExternalAuthor: !feedauthor.IsOwnPost(sa.Identity, p.AuthorHandle, p.AuthorName),

		InteractionRemoteId: p.InteractionRemoteID,
		Liked:               p.Liked,
		LikeCount:           p.LikeCount,
		ReplyCount:          p.ReplyCount,
		CanLike:             s.connectorSupportsLikes(sa.Connector),
		CanViewComments:     s.connectorSupportsReplies(sa.Connector),
		CanComment:          s.connectorSupportsReplying(sa.Connector),
		CanRefresh:          true,
	}
	if !p.PostedDate.IsZero() {
		out.PostedDate = p.PostedDate.Local().Format("2006-01-02 15:04:05")
	}
	return out
}

// RefreshPost reloads a post from its network: content, likes and comment
// counts. Stored feed posts are updated in place, and a tagged post's copy
// is refreshed too.
func (s *ControlApi) RefreshPost(ctx context.Context, req *connect.Request[controlv1.RefreshPostRequest]) (*connect.Response[controlv1.RefreshPostResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if s.cc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("service initializing"))
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
	rc, ok := svc.(connector.ConnectorWithFeedPostRefetch)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("refreshing posts isn't available for %s", sa.Connector))
	}
	sa = s.freshSocialAccount(sa, svc)

	var post *controlv1.FeedPost
	if id := ref.GetFeedPostId(); id != 0 {
		entry, err := s.DB.GetFeedEntryByID(id)
		if err != nil || entry == nil || entry.SocialAccountID != sa.ID {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("feed post not found"))
		}
		if err := s.refetchFeedEntry(rc, sa, entry); err != nil {
			return nil, refreshError(sa, err)
		}
		if err := s.DB.UpdateFeedEntry(entry); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update feed post"))
		}
		post = s.feedEntryToProto(entry)
	} else {
		fetched, err := rc.FetchFeedPost(toConnectorSA(sa), ref.GetRemoteId(), ref.GetRemoteUrl())
		if err != nil {
			return nil, refreshError(sa, err)
		}
		post = s.fetchedPostToProto(fetched, sa)
		// Keep the identity the client knows the post by.
		if ref.GetRemoteUrl() != "" || ref.GetRemoteId() != "" {
			post.RemoteUrl, post.RemoteId = ref.GetRemoteUrl(), ref.GetRemoteId()
		}
	}

	if key := feedauthor.PostKey(post.RemoteUrl, post.RemoteId); key != "" {
		hash := feedauthor.KeyHash(key)
		commented, _ := s.DB.SelectCommentedPostHashes([]uint32{sa.ID})
		post.Commented = commented[sa.ID][hash]
		if tags := s.userPostTags(au)[hash]; len(tags) > 0 {
			post.Tags = tags
			_ = s.saveTaggedPost(au.User.ID, key, sa.ID, post, tags)
		}
	}

	return connect.NewResponse(&controlv1.RefreshPostResponse{Post: post}), nil
}

func refreshError(sa *db.SocialAccount, err error) error {
	if rl := connector.AsRateLimited(err); rl != nil {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited by %s; try again in %s", sa.Connector, rl.RetryAfter))
	}
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to refresh post from %s: %w", sa.Connector, err))
}
