package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/posttags"
	"github.com/jamesread/japella/internal/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const tagColumnPostLimit = 100

// safeSnapshotURL keeps only http(s) URLs, since the UI uses them as links and images.
func safeSnapshotURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > 2048 {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return raw
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// clientTagSnapshot copies the displayable fields of a post the client sent.
// Column posts are not stored server side, so this is the only copy; the
// content is shown as HTML, so it is reduced to escaped text.
func clientTagSnapshot(in *controlv1.FeedPost, ref *controlv1.PostRef) *controlv1.FeedPost {
	if in == nil {
		in = &controlv1.FeedPost{}
	}
	text := func(s string) string { return truncateRunes(strings.TrimSpace(s), 255) }
	return &controlv1.FeedPost{
		Content:                  utils.TextToSafeHTML(truncateRunes(utils.HTMLToText(in.GetContent()), 20000)),
		PostedDate:               text(in.GetPostedDate()),
		AuthorId:                 text(in.GetAuthorId()),
		AuthorName:               text(in.GetAuthorName()),
		AuthorAvatarUrl:          safeSnapshotURL(in.GetAuthorAvatarUrl()),
		AuthorHandle:             text(in.GetAuthorHandle()),
		AuthorProfileUrl:         safeSnapshotURL(in.GetAuthorProfileUrl()),
		OriginalAuthorId:         text(in.GetOriginalAuthorId()),
		OriginalAuthorName:       text(in.GetOriginalAuthorName()),
		OriginalAuthorAvatarUrl:  safeSnapshotURL(in.GetOriginalAuthorAvatarUrl()),
		OriginalAuthorHandle:     text(in.GetOriginalAuthorHandle()),
		OriginalAuthorProfileUrl: safeSnapshotURL(in.GetOriginalAuthorProfileUrl()),
		RemoteUrl:                safeSnapshotURL(ref.GetRemoteUrl()),
		RemoteId:                 truncateRunes(ref.GetRemoteId(), 1024),
		InteractionRemoteId:      truncateRunes(ref.GetInteractionRemoteId(), 1024),
		PreviewUrl:               safeSnapshotURL(in.GetPreviewUrl()),
		PreviewTitle:             text(in.GetPreviewTitle()),
		PreviewDescription:       utils.HTMLToText(truncateRunes(in.GetPreviewDescription(), 1000)),
		PreviewImageUrl:          safeSnapshotURL(in.GetPreviewImageUrl()),
		Language:                 text(in.GetLanguage()),
		Liked:                    in.GetLiked(),
		LikeCount:                in.GetLikeCount(),
		ReplyCount:               in.GetReplyCount(),
	}
}

// tagSnapshot returns the post to remember and its platform key.
func (s *ControlApi) tagSnapshot(ref *controlv1.PostRef, sa *db.SocialAccount, sent *controlv1.FeedPost) (*controlv1.FeedPost, string, error) {
	if id := ref.GetFeedPostId(); id != 0 {
		entry, err := s.DB.GetFeedEntryByID(id)
		if err != nil || entry == nil || entry.SocialAccountID != sa.ID {
			return nil, "", connect.NewError(connect.CodeNotFound, fmt.Errorf("feed post not found"))
		}
		return s.feedEntryToProto(entry), feedauthor.PostKey(entry.RemoteURL, entry.RemoteID), nil
	}
	snap := clientTagSnapshot(sent, ref)
	return snap, feedauthor.PostKey(ref.GetRemoteUrl(), ref.GetRemoteId()), nil
}

// SetPostTags replaces the caller's internal tags on a post. Tags are private
// to the user and nothing is sent to the post's network.
func (s *ControlApi) SetPostTags(ctx context.Context, req *connect.Request[controlv1.SetPostTagsRequest]) (*connect.Response[controlv1.SetPostTagsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	tags, err := posttags.NormaliseList(req.Msg.GetTags())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ref := req.Msg.GetPost()
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanReadSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}

	snap, key, err := s.tagSnapshot(ref, sa, req.Msg.GetSnapshot())
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post remote id or url is required"))
	}

	if err := s.saveTaggedPost(au.User.ID, key, sa.ID, snap, tags); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save tags"))
	}
	return connect.NewResponse(&controlv1.SetPostTagsResponse{Tags: tags}), nil
}

// saveTaggedPost stores a post's tags with a copy of the post.
func (s *ControlApi) saveTaggedPost(userID uint32, key string, socialAccountID uint32, post *controlv1.FeedPost, tags []string) error {
	snap := proto.Clone(post).(*controlv1.FeedPost)
	// Per-viewer state is recomputed when the post is shown.
	snap.Id, snap.Tags, snap.Commented = 0, nil, false
	snap.CanLike, snap.CanViewComments, snap.CanComment, snap.CanRefresh = false, false, false, false
	raw, err := protojson.Marshal(snap)
	if err != nil {
		return err
	}
	return s.DB.SetPostTags(userID, feedauthor.KeyHash(key), key, socialAccountID, string(raw), tags)
}

func (s *ControlApi) ListPostTags(ctx context.Context, req *connect.Request[controlv1.ListPostTagsRequest]) (*connect.Response[controlv1.ListPostTagsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	counts, err := s.DB.SelectTagCounts(au.User.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load tags"))
	}
	res := &controlv1.ListPostTagsResponse{}
	for _, c := range counts {
		res.Tags = append(res.Tags, &controlv1.PostTagCount{Tag: c.Tag, Posts: c.Posts})
	}
	return connect.NewResponse(res), nil
}

// userPostTags returns the caller's tags by post key hash; empty without a user.
func (s *ControlApi) userPostTags(au *authentication.AuthenticatedUser) map[string][]string {
	if au == nil || au.User == nil {
		return map[string][]string{}
	}
	tags, _ := s.DB.SelectPostTagsByHash(au.User.ID)
	return tags
}

// tagColumnPosts lists the caller's posts with tag, from their snapshots.
func (s *ControlApi) tagColumnPosts(au *authentication.AuthenticatedUser, tag string) (*controlv1.GetFeedColumnPostsResponse, error) {
	tagged, err := s.DB.SelectTaggedPosts(au.User.ID, tag, tagColumnPostLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load tagged posts"))
	}
	tagsByHash := s.userPostTags(au)

	accountIDs := make([]uint32, 0)
	accounts := map[uint32]*db.SocialAccount{}
	readable := map[uint32]bool{}
	for _, tp := range tagged {
		if _, ok := readable[tp.SocialAccountID]; ok {
			continue
		}
		readable[tp.SocialAccountID] = s.userCanReadSocialAccount(au, tp.SocialAccountID)
		if readable[tp.SocialAccountID] {
			accounts[tp.SocialAccountID], _ = s.DB.GetSocialAccount(tp.SocialAccountID)
			accountIDs = append(accountIDs, tp.SocialAccountID)
		}
	}
	commented, _ := s.DB.SelectCommentedPostHashes(accountIDs)

	res := &controlv1.GetFeedColumnPostsResponse{}
	for _, tp := range tagged {
		sa := accounts[tp.SocialAccountID]
		// Sharing may have been revoked since the post was tagged.
		if !readable[tp.SocialAccountID] || sa == nil {
			continue
		}
		post := &controlv1.FeedPost{}
		if err := protojson.Unmarshal([]byte(tp.Snapshot), post); err != nil {
			continue
		}
		// Stored feed posts get pruned, so tag column posts are always
		// addressed by their platform identity.
		post.Id = 0
		post.SocialAccountId = sa.ID
		post.SocialAccountIdentity = sa.Identity
		post.SocialAccountConnector = sa.Connector
		post.SocialAccountIcon = "mdi:question-mark-circle"
		if svc := s.cc.Get(sa.Connector); svc != nil {
			post.SocialAccountIcon = svc.GetIcon()
		}
		post.FromExternalAuthor = !feedauthor.IsOwnPost(sa.Identity, post.AuthorHandle, post.AuthorName)
		post.CanLike = s.connectorSupportsLikes(sa.Connector)
		post.CanViewComments = s.connectorSupportsReplies(sa.Connector)
		post.CanComment = s.connectorSupportsReplying(sa.Connector)
		post.CanRefresh = s.connectorSupportsRefetch(sa.Connector)
		post.Commented = commented[sa.ID][tp.PostKeyHash]
		post.Tags = tagsByHash[tp.PostKeyHash]
		res.Posts = append(res.Posts, post)
	}
	return res, nil
}
