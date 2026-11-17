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

func isHTTPSnapshotURL(u *url.URL, err error) bool {
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// safeSnapshotURL keeps only http(s) URLs, since the UI uses them as links and images.
func safeSnapshotURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > 2048 {
		return ""
	}
	u, err := url.Parse(raw)
	if !isHTTPSnapshotURL(u, err) {
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
func (s *ControlApi) authorizeTaggablePost(au *authentication.AuthenticatedUser, ref *controlv1.PostRef) (*db.SocialAccount, error) {
	sa, err := s.postRefAccount(ref)
	if err != nil {
		return nil, err
	}
	if !s.userCanReadSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}
	return sa, nil
}

func (s *ControlApi) requireTagSnapshot(ref *controlv1.PostRef, sa *db.SocialAccount, sent *controlv1.FeedPost) (*controlv1.FeedPost, string, error) {
	snap, key, err := s.tagSnapshot(ref, sa, sent)
	if err != nil {
		return nil, "", err
	}
	if key == "" {
		return nil, "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post remote id or url is required"))
	}
	return snap, key, nil
}

func (s *ControlApi) prepareSetPostTags(au *authentication.AuthenticatedUser, msg *controlv1.SetPostTagsRequest) ([]string, *db.SocialAccount, error) {
	tags, err := posttags.NormaliseList(msg.GetTags())
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	sa, err := s.authorizeTaggablePost(au, msg.GetPost())
	if err != nil {
		return nil, nil, err
	}
	return tags, sa, nil
}

func (s *ControlApi) SetPostTags(ctx context.Context, req *connect.Request[controlv1.SetPostTagsRequest]) (*connect.Response[controlv1.SetPostTagsResponse], error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}
	tags, sa, err := s.prepareSetPostTags(au, req.Msg)
	if err != nil {
		return nil, err
	}
	snap, key, err := s.requireTagSnapshot(req.Msg.GetPost(), sa, req.Msg.GetSnapshot())
	if err != nil {
		return nil, err
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
func (s *ControlApi) tagColumnReadableAccounts(au *authentication.AuthenticatedUser, tagged []*db.TaggedPost) (map[uint32]*db.SocialAccount, map[uint32]bool, []uint32) {
	accounts := map[uint32]*db.SocialAccount{}
	readable := map[uint32]bool{}
	accountIDs := make([]uint32, 0)
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
	return accounts, readable, accountIDs
}

func (s *ControlApi) hydrateTaggedColumnPost(tp *db.TaggedPost, sa *db.SocialAccount, commented map[uint32]map[string]bool, tagsByHash map[string][]string) *controlv1.FeedPost {
	post := &controlv1.FeedPost{}
	if err := protojson.Unmarshal([]byte(tp.Snapshot), post); err != nil {
		return nil
	}
	post.Id = 0
	post.SocialAccountId = sa.ID
	post.SocialAccountIdentity = sa.Identity
	post.SocialAccountConnector = sa.Connector
	post.SocialAccountIcon = s.connectorIcon(sa.Connector)
	post.FromExternalAuthor = !feedauthor.IsOwnPost(sa.Identity, post.AuthorHandle, post.AuthorName)
	post.CanLike = s.connectorSupportsLikes(sa.Connector)
	post.CanViewComments = s.connectorSupportsReplies(sa.Connector)
	post.CanComment = s.connectorSupportsReplying(sa.Connector)
	post.CanRefresh = s.connectorSupportsRefetch(sa.Connector)
	post.Commented = commented[sa.ID][tp.PostKeyHash]
	post.Tags = tagsByHash[tp.PostKeyHash]
	return post
}

func (s *ControlApi) taggedColumnPostOrNil(tp *db.TaggedPost, accounts map[uint32]*db.SocialAccount, readable map[uint32]bool, commented map[uint32]map[string]bool, tagsByHash map[string][]string) *controlv1.FeedPost {
	sa := accounts[tp.SocialAccountID]
	if !readable[tp.SocialAccountID] || sa == nil {
		return nil
	}
	return s.hydrateTaggedColumnPost(tp, sa, commented, tagsByHash)
}

func (s *ControlApi) tagColumnPosts(au *authentication.AuthenticatedUser, tag string) (*controlv1.GetFeedColumnPostsResponse, error) {
	tagged, err := s.DB.SelectTaggedPosts(au.User.ID, tag, tagColumnPostLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load tagged posts"))
	}
	tagsByHash := s.userPostTags(au)
	accounts, readable, accountIDs := s.tagColumnReadableAccounts(au, tagged)
	commented, _ := s.DB.SelectCommentedPostHashes(accountIDs)
	res := &controlv1.GetFeedColumnPostsResponse{}
	for _, tp := range tagged {
		if post := s.taggedColumnPostOrNil(tp, accounts, readable, commented, tagsByHash); post != nil {
			res.Posts = append(res.Posts, post)
		}
	}
	return res, nil
}
