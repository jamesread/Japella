package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/feedcolumns"
	"github.com/jamesread/japella/internal/languages"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/utils"
)

const (
	feedColumnsMaxPerUser = 10
	feedColumnPostLimit   = 30
	feedColumnTitleMax    = 255
	feedColumnMaxAccounts = 10
)

// feedColumnCache is shared by all users; entries are keyed by column id.
var feedColumnCache = feedcolumns.NewCache(time.Minute)

func feedColumnUser(au *authentication.AuthenticatedUser) (uint32, error) {
	if au == nil || au.User == nil {
		return 0, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return au.User.ID, nil
}

func (s *ControlApi) feedColumnSearch(connectorName string) (connector.ConnectorWithKeywordSearch, connector.BaseConnector) {
	if s.cc == nil {
		return nil, nil
	}
	svc := s.cc.Get(connectorName)
	if svc == nil {
		return nil, nil
	}
	search, _ := svc.(connector.ConnectorWithKeywordSearch)
	return search, svc
}

// feedColumnAccounts loads a column's accounts, skipping any that no longer exist.
func (s *ControlApi) feedColumnAccounts(ids []uint32, cache map[uint32]*db.SocialAccount) []*db.SocialAccount {
	out := make([]*db.SocialAccount, 0, len(ids))
	for _, id := range ids {
		sa, ok := cache[id]
		if !ok {
			sa, _ = s.DB.GetSocialAccount(id)
			cache[id] = sa
		}
		if sa != nil {
			out = append(out, sa)
		}
	}
	return out
}

func (s *ControlApi) feedColumnToProto(col *db.FeedColumn, accounts []*db.SocialAccount) *controlv1.FeedColumn {
	q, _ := feedcolumns.ParseQuery(col.Query)
	out := &controlv1.FeedColumn{
		Id:        col.ID,
		Title:     col.Title,
		Query:     col.Query,
		IsHashtag: q.Hashtag != "",
		Tag:       q.Tag,
	}
	if out.Title == "" {
		out.Title = feedcolumns.DefaultTitle(q)
	}
	for _, sa := range accounts {
		acc := &controlv1.FeedColumnAccount{
			Id:        sa.ID,
			Identity:  sa.Identity,
			Connector: sa.Connector,
			Icon:      "mdi:question-mark-circle",
		}
		search, svc := s.feedColumnSearch(sa.Connector)
		acc.Supported = search != nil
		if svc != nil {
			acc.Icon = svc.GetIcon()
		}
		out.Accounts = append(out.Accounts, acc)
	}
	return out
}

// validateFeedColumnInput checks the query, title and that the user may read every account.
func (s *ControlApi) validateFeedColumnInput(au *authentication.AuthenticatedUser, rawQuery, rawTitle string, socialAccountIDs []uint32) (*db.FeedColumn, []*db.SocialAccount, error) {
	q, err := feedcolumns.ParseQuery(rawQuery)
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	title := strings.TrimSpace(rawTitle)
	if utf8.RuneCountInString(title) > feedColumnTitleMax {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title must be at most %d characters", feedColumnTitleMax))
	}

	// Tag columns read the user's own tagged posts, not any network.
	if q.Tag != "" {
		return &db.FeedColumn{Title: title, Query: q.Text}, nil, nil
	}

	ids := make([]uint32, 0, len(socialAccountIDs))
	seen := map[uint32]bool{}
	for _, id := range socialAccountIDs {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("choose at least one social account"))
	}
	if len(ids) > feedColumnMaxAccounts {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("a column can search at most %d accounts", feedColumnMaxAccounts))
	}

	accounts := make([]*db.SocialAccount, 0, len(ids))
	for _, id := range ids {
		if !s.userCanReadSocialAccount(au, id) {
			return nil, nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to social account %d is required", id))
		}
		sa, err := s.DB.GetSocialAccount(id)
		if err != nil || sa == nil {
			return nil, nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account %d not found", id))
		}
		accounts = append(accounts, sa)
	}

	return &db.FeedColumn{SocialAccountIDs: ids, Title: title, Query: q.Text}, accounts, nil
}

// ownFeedColumn loads a column, hiding other users' columns as not found.
func (s *ControlApi) ownFeedColumn(userID, id uint32) (*db.FeedColumn, error) {
	if id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	col, err := s.DB.GetFeedColumn(id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load feed column"))
	}
	if col == nil || col.UserAccountID != userID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("feed column not found"))
	}
	return col, nil
}

func (s *ControlApi) ListFeedColumns(ctx context.Context, req *connect.Request[controlv1.ListFeedColumnsRequest]) (*connect.Response[controlv1.ListFeedColumnsResponse], error) {
	userID, err := feedColumnUser(s.getAuthenticatedUser(ctx))
	if err != nil {
		return nil, err
	}
	cols, err := s.DB.SelectFeedColumnsForUser(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load feed columns"))
	}

	cache := map[uint32]*db.SocialAccount{}
	res := &controlv1.ListFeedColumnsResponse{}
	for _, col := range cols {
		res.Columns = append(res.Columns, s.feedColumnToProto(col, s.feedColumnAccounts(col.SocialAccountIDs, cache)))
	}
	return connect.NewResponse(res), nil
}

func (s *ControlApi) CreateFeedColumn(ctx context.Context, req *connect.Request[controlv1.CreateFeedColumnRequest]) (*connect.Response[controlv1.CreateFeedColumnResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	userID, err := feedColumnUser(au)
	if err != nil {
		return nil, err
	}

	existing, err := s.DB.SelectFeedColumnsForUser(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load feed columns"))
	}
	if len(existing) >= feedColumnsMaxPerUser {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("at most %d feed columns are allowed", feedColumnsMaxPerUser))
	}

	col, accounts, err := s.validateFeedColumnInput(au, req.Msg.GetQuery(), req.Msg.GetTitle(), req.Msg.GetSocialAccountIds())
	if err != nil {
		return nil, err
	}
	col.UserAccountID = userID

	id, err := s.DB.CreateFeedColumn(col)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create feed column"))
	}
	col.ID = id

	return connect.NewResponse(&controlv1.CreateFeedColumnResponse{Column: s.feedColumnToProto(col, accounts)}), nil
}

func (s *ControlApi) UpdateFeedColumn(ctx context.Context, req *connect.Request[controlv1.UpdateFeedColumnRequest]) (*connect.Response[controlv1.UpdateFeedColumnResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	userID, err := feedColumnUser(au)
	if err != nil {
		return nil, err
	}
	existing, err := s.ownFeedColumn(userID, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	col, accounts, err := s.validateFeedColumnInput(au, req.Msg.GetQuery(), req.Msg.GetTitle(), req.Msg.GetSocialAccountIds())
	if err != nil {
		return nil, err
	}
	col.ID = existing.ID
	col.UserAccountID = userID
	col.SortOrder = existing.SortOrder

	if err := s.DB.UpdateFeedColumn(col); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update feed column"))
	}
	feedColumnCache.Forget(col.ID)

	return connect.NewResponse(&controlv1.UpdateFeedColumnResponse{Column: s.feedColumnToProto(col, accounts)}), nil
}

func (s *ControlApi) DeleteFeedColumn(ctx context.Context, req *connect.Request[controlv1.DeleteFeedColumnRequest]) (*connect.Response[controlv1.DeleteFeedColumnResponse], error) {
	userID, err := feedColumnUser(s.getAuthenticatedUser(ctx))
	if err != nil {
		return nil, err
	}
	if _, err := s.ownFeedColumn(userID, req.Msg.GetId()); err != nil {
		return nil, err
	}
	if err := s.DB.DeleteFeedColumn(userID, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete feed column"))
	}
	feedColumnCache.Forget(req.Msg.GetId())
	return connect.NewResponse(&controlv1.DeleteFeedColumnResponse{}), nil
}

func (s *ControlApi) ReorderFeedColumns(ctx context.Context, req *connect.Request[controlv1.ReorderFeedColumnsRequest]) (*connect.Response[controlv1.ReorderFeedColumnsResponse], error) {
	userID, err := feedColumnUser(s.getAuthenticatedUser(ctx))
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetIds()) > feedColumnsMaxPerUser {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("too many ids"))
	}
	// Updates are scoped to the caller's own columns, so foreign ids are ignored.
	if err := s.DB.ReorderFeedColumns(userID, req.Msg.GetIds()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reorder feed columns"))
	}
	return connect.NewResponse(&controlv1.ReorderFeedColumnsResponse{}), nil
}

// filterByLanguage drops posts outside an account's preferred languages.
func filterByLanguage(posts []*connector.RemotePost, filter languages.Filter) []*connector.RemotePost {
	out := make([]*connector.RemotePost, 0, len(posts))
	for _, p := range posts {
		if p != nil && filter.Allows(p.Language) {
			out = append(out, p)
		}
	}
	return out
}

// filterUnread drops posts the user has marked read.
func filterUnread(posts []*connector.RemotePost, read map[string]bool) []*connector.RemotePost {
	if len(read) == 0 {
		return posts
	}
	out := make([]*connector.RemotePost, 0, len(posts))
	for _, p := range posts {
		if key := feedauthor.PostKey(p.RemoteURL, p.RemoteID); key == "" || !read[feedauthor.KeyHash(key)] {
			out = append(out, p)
		}
	}
	return out
}

// feedColumnPostToProto renders a search hit like a feed post. Content comes
// from strangers and the UI shows it as HTML, so it is always reduced to
// escaped text here.
func feedColumnPostToProto(p *connector.RemotePost, sa *db.SocialAccount, icon string) *controlv1.FeedPost {
	text := p.Content
	if p.ContentIsHTML {
		text = utils.HTMLToText(text)
	}
	author := p.AuthorName
	if author == "" {
		author = p.AuthorHandle
	}
	out := &controlv1.FeedPost{
		SocialAccountId:        sa.ID,
		SocialAccountIcon:      icon,
		SocialAccountIdentity:  sa.Identity,
		SocialAccountConnector: sa.Connector,
		Content:                utils.TextToSafeHTML(text),
		AuthorId:               p.AuthorID,
		AuthorName:             author,
		AuthorAvatarUrl:        p.AuthorAvatarURL,
		RemoteUrl:              p.RemoteURL,
		RemoteId:               p.RemoteID,
		Language:               p.Language,
		AuthorHandle:           p.AuthorHandle,
		AuthorProfileUrl:       p.AuthorProfileURL,

		FromExternalAuthor: !feedauthor.IsOwnPost(sa.Identity, p.AuthorHandle, p.AuthorName),

		InteractionRemoteId: p.InteractionRemoteID,
		Liked:               p.Liked,
		LikeCount:           p.LikeCount,
		ReplyCount:          p.ReplyCount,
	}
	if !p.PostedAt.IsZero() {
		out.PostedDate = p.PostedAt.Local().Format("2006-01-02 15:04:05")
	}
	return out
}

// fetchFeedColumnAccount returns one account's (cached) results for a column.
func (s *ControlApi) fetchFeedColumnAccount(ctx context.Context, col *db.FeedColumn, q feedcolumns.Query, sa *db.SocialAccount) feedcolumns.Result {
	search, svc := s.feedColumnSearch(sa.Connector)
	if search == nil {
		return feedcolumns.Result{Err: fmt.Errorf("search isn't available for this network")}
	}

	return feedColumnCache.Get(col.ID, sa.ID, q.Text, time.Now(), func() (posts []*connector.RemotePost, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("search failed: %v", r)
			}
		}()

		if socialAccountNeedsTokenRefresh(sa, time.Hour) {
			if refreshed, refreshErr := s.refreshSocialAccountToken(sa, svc); refreshErr == nil {
				sa = refreshed
			}
		}

		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if tl, ok := svc.(connector.ConnectorWithHashtagTimeline); ok && q.Hashtag != "" {
			return tl.FetchHashtagTimeline(fetchCtx, toConnectorSA(sa), q.Hashtag, feedColumnPostLimit)
		}
		return search.SearchRecentByKeyword(fetchCtx, toConnectorSA(sa), q.Text, time.Time{}, feedColumnPostLimit)
	})
}

// GetFeedColumnPosts searches every account on a column (in parallel) and
// merges the results, newest first.
func (s *ControlApi) GetFeedColumnPosts(ctx context.Context, req *connect.Request[controlv1.GetFeedColumnPostsRequest]) (*connect.Response[controlv1.GetFeedColumnPostsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	userID, err := feedColumnUser(au)
	if err != nil {
		return nil, err
	}
	col, err := s.ownFeedColumn(userID, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	q, err := feedcolumns.ParseQuery(col.Query)
	if err != nil {
		return connect.NewResponse(&controlv1.GetFeedColumnPostsResponse{Error: err.Error()}), nil
	}
	if q.Tag != "" {
		res, err := s.tagColumnPosts(au, q.Tag)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(res), nil
	}

	// Re-check access: sharing may have been revoked since the column was saved.
	accounts := make([]*db.SocialAccount, 0, len(col.SocialAccountIDs))
	for _, sa := range s.feedColumnAccounts(col.SocialAccountIDs, map[uint32]*db.SocialAccount{}) {
		if s.userCanReadSocialAccount(au, sa.ID) {
			accounts = append(accounts, sa)
		}
	}
	if len(accounts) == 0 {
		return connect.NewResponse(&controlv1.GetFeedColumnPostsResponse{Error: "None of this column's accounts are available to you any more. Edit the column to choose another."}), nil
	}

	readRemote := map[string]bool{}
	if au.User != nil {
		if readRemote, err = s.DB.SelectReadRemotePostHashes(au.User.ID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve read posts"))
		}
	}

	accountIDs := make([]uint32, len(accounts))
	for i, sa := range accounts {
		accountIDs[i] = sa.ID
	}
	commented, _ := s.DB.SelectCommentedPostHashes(accountIDs)
	tagsByHash := s.userPostTags(au)

	results := make([]feedcolumns.Result, len(accounts))
	var wg sync.WaitGroup
	for i, sa := range accounts {
		wg.Add(1)
		go func(i int, sa *db.SocialAccount) {
			defer wg.Done()
			results[i] = s.fetchFeedColumnAccount(ctx, col, q, sa)
		}(i, sa)
	}
	wg.Wait()

	res := &controlv1.GetFeedColumnPostsResponse{}
	byAccount := make(map[uint32][]*connector.RemotePost, len(accounts))
	order := make([]uint32, 0, len(accounts))
	accountByID := make(map[uint32]*db.SocialAccount, len(accounts))
	var failures []string
	var oldest time.Time

	for i, sa := range accounts {
		r := results[i]
		accountByID[sa.ID] = sa
		order = append(order, sa.ID)
		byAccount[sa.ID] = filterUnread(filterByLanguage(r.Posts, languages.NewFilter(sa.PreferredLanguages)), readRemote)
		if !r.FetchedAt.IsZero() && (oldest.IsZero() || r.FetchedAt.Before(oldest)) {
			oldest = r.FetchedAt
		}
		if r.Err != nil {
			msg := r.Err.Error()
			if !r.RetryAfter.IsZero() {
				msg = "rate limited, retrying later"
			}
			failures = append(failures, sa.Identity+": "+msg)
		}
	}

	if len(failures) > 0 {
		res.Error = "Search failed for " + strings.Join(failures, "; ")
	}
	if !oldest.IsZero() {
		res.FetchedAt = oldest.UTC().Format(time.RFC3339)
	}

	for _, sp := range feedcolumns.Merge(byAccount, order, feedColumnPostLimit) {
		sa := accountByID[sp.AccountID]
		icon := "mdi:question-mark-circle"
		if _, svc := s.feedColumnSearch(sa.Connector); svc != nil {
			icon = svc.GetIcon()
		}
		post := feedColumnPostToProto(sp.Post, sa, icon)
		post.CanLike = s.connectorSupportsLikes(sa.Connector)
		post.CanViewComments = s.connectorSupportsReplies(sa.Connector)
		post.CanComment = s.connectorSupportsReplying(sa.Connector)
		post.CanRefresh = s.connectorSupportsRefetch(sa.Connector)
		if key := feedauthor.PostKey(sp.Post.RemoteURL, sp.Post.RemoteID); key != "" {
			post.Commented = commented[sa.ID][feedauthor.KeyHash(key)]
			post.Tags = tagsByHash[feedauthor.KeyHash(key)]
		}
		res.Posts = append(res.Posts, post)
	}
	return connect.NewResponse(res), nil
}
