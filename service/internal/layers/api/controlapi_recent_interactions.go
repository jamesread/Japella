package api

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/keywordwatch"
	"github.com/jamesread/japella/internal/postengagement"
	"github.com/jamesread/japella/internal/postreplies"
)

const (
	recentInteractionsDefaultDays = 7
	recentInteractionsMaxDays     = 30
	recentInteractionsMaxItems    = 200
	recentInteractionSnippetRunes = 120
)

func postSnippet(content string) string {
	text := keywordwatch.StripHTML(content)
	if utf8.RuneCountInString(text) <= recentInteractionSnippetRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:recentInteractionSnippetRunes-1]) + "…"
}

var recentInteractionFillers = map[string]func(*controlv1.RecentInteraction, postengagement.Interaction){
	postengagement.InteractionEngagement: func(out *controlv1.RecentInteraction, it postengagement.Interaction) {
		out.Engagement = countsToProto(it.Counts)
	},
	postengagement.InteractionReply: func(out *controlv1.RecentInteraction, it postengagement.Interaction) {
		if replies := replyNodesToProto([]*postreplies.Node{{Reply: it.Reply}}); len(replies) == 1 {
			out.Reply = replies[0]
		}
	},
	postengagement.InteractionLiked: func(out *controlv1.RecentInteraction, it postengagement.Interaction) {
		out.PostAuthor = it.PostAuthor
	},
	postengagement.InteractionCommented: func(out *controlv1.RecentInteraction, it postengagement.Interaction) {
		out.PostAuthor = it.PostAuthor
		out.Comment = it.Comment
		out.CommentUrl = it.CommentURL
	},
}

func recentInteractionToProto(it postengagement.Interaction) *controlv1.RecentInteraction {
	out := &controlv1.RecentInteraction{
		Kind:        it.Kind,
		At:          it.At.UTC().Format(time.RFC3339),
		PostId:      it.PostID,
		PostSnippet: postSnippet(it.PostContent),
		PostUrl:     it.PostURL,
	}
	if fill, ok := recentInteractionFillers[it.Kind]; ok {
		fill(out, it)
	}
	return out
}

func recentInteractionsDays(requested uint32) int {
	days := recentInteractionsDefaultDays
	if d := int(requested); d > 0 {
		days = min(d, recentInteractionsMaxDays)
	}
	return days
}

func (s *ControlApi) loadRecentInteractionSources(accountID uint32, since time.Time, window time.Duration) (postengagement.InteractionSources, error) {
	posts, err := s.DB.SelectCompletedPostsForAccountWithin(accountID, window)
	if err != nil {
		return postengagement.InteractionSources{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load posts"))
	}
	events, err := s.DB.SelectPostEngagementEventsForAccountSince(accountID, since)
	if err != nil {
		return postengagement.InteractionSources{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load engagement"))
	}
	replies, err := s.DB.SelectPostRepliesForAccountSince(accountID, since)
	if err != nil {
		return postengagement.InteractionSources{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load replies"))
	}
	activity, err := s.DB.SelectAccountActivitySince(accountID, since)
	if err != nil {
		return postengagement.InteractionSources{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load likes and comments"))
	}
	return postengagement.InteractionSources{
		Posts:    posts,
		Replies:  replies,
		Events:   events,
		Activity: activity,
	}, nil
}

func (s *ControlApi) requireReadableSocialAccount(ctx context.Context, accountID uint32) (*db.SocialAccount, error) {
	au, err := s.requireAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	if accountID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id is required"))
	}
	if !s.userCanReadSocialAccount(au, accountID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}
	return s.loadSocialAccountOrErr(accountID)
}

func (s *ControlApi) loadSocialAccountOrErr(accountID uint32) (*db.SocialAccount, error) {
	sa, err := s.DB.GetSocialAccount(accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load social account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	return sa, nil
}

// GetSocialAccountRecentInteractions lists what happened on an account over
// the last N days: posts it published, replies it received, increases in
// engagement, and likes and comments it made, newest first, with totals and
// per-day counts.
func (s *ControlApi) GetSocialAccountRecentInteractions(ctx context.Context, req *connect.Request[controlv1.GetSocialAccountRecentInteractionsRequest]) (*connect.Response[controlv1.GetSocialAccountRecentInteractionsResponse], error) {
	sa, err := s.requireReadableSocialAccount(ctx, req.Msg.GetSocialAccountId())
	if err != nil {
		return nil, err
	}

	days := recentInteractionsDays(req.Msg.GetDays())
	window := time.Duration(days) * 24 * time.Hour
	since := time.Now().UTC().Add(-window)

	sources, err := s.loadRecentInteractionSources(sa.ID, since, window)
	if err != nil {
		return nil, err
	}
	sources.RepliesFromThreads = s.connectorSupportsReplies(sa.Connector)

	sum := postengagement.SummariseInteractions(time.Now(), days, sources, recentInteractionsMaxItems)
	return connect.NewResponse(s.recentInteractionsResponse(sa, days, sum)), nil
}

func (s *ControlApi) recentInteractionsResponse(sa *db.SocialAccount, days int, sum postengagement.InteractionSummary) *controlv1.GetSocialAccountRecentInteractionsResponse {
	res := &controlv1.GetSocialAccountRecentInteractionsResponse{
		Days:            intconv.ToUint32(days),
		Supported:       s.connectorSupportsEngagement(sa.Connector),
		TrackingEnabled: s.postEngagementMaxAge() > 0,
		Posts:           sum.Posts,
		Totals:          countsToProto(sum.Counts),
		LikesGiven:      sum.LikesGiven,
		CommentsMade:    sum.CommentsMade,
	}
	for _, b := range sum.Daily {
		res.Daily = append(res.Daily, &controlv1.PostPerformanceDay{
			Date:         b.Date,
			Posts:        b.Posts,
			Engagement:   countsToProto(b.Counts),
			LikesGiven:   b.LikesGiven,
			CommentsMade: b.CommentsMade,
		})
	}
	for _, it := range sum.Items {
		res.Items = append(res.Items, recentInteractionToProto(it))
	}
	return res
}
