package api

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
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

func recentInteractionToProto(it postengagement.Interaction) *controlv1.RecentInteraction {
	out := &controlv1.RecentInteraction{
		Kind:        it.Kind,
		At:          it.At.UTC().Format(time.RFC3339),
		PostId:      it.PostID,
		PostSnippet: postSnippet(it.PostContent),
		PostUrl:     it.PostURL,
	}
	switch it.Kind {
	case postengagement.InteractionEngagement:
		out.Engagement = countsToProto(it.Counts)
	case postengagement.InteractionReply:
		if replies := replyNodesToProto([]*postreplies.Node{{Reply: it.Reply}}); len(replies) == 1 {
			out.Reply = replies[0]
		}
	}
	return out
}

// GetSocialAccountRecentInteractions lists what happened on an account over
// the last N days: posts it published, replies it received and increases in
// engagement, newest first, with totals and per-day counts.
func (s *ControlApi) GetSocialAccountRecentInteractions(ctx context.Context, req *connect.Request[controlv1.GetSocialAccountRecentInteractionsRequest]) (*connect.Response[controlv1.GetSocialAccountRecentInteractionsResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	accountID := req.Msg.GetSocialAccountId()
	if accountID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id is required"))
	}
	if !s.userCanReadSocialAccount(au, accountID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}

	sa, err := s.DB.GetSocialAccount(accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load social account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}

	days := recentInteractionsDefaultDays
	if d := int(req.Msg.GetDays()); d > 0 {
		days = min(d, recentInteractionsMaxDays)
	}
	window := time.Duration(days) * 24 * time.Hour
	since := time.Now().UTC().Add(-window)

	posts, err := s.DB.SelectCompletedPostsForAccountWithin(accountID, window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load posts"))
	}
	events, err := s.DB.SelectPostEngagementEventsForAccountSince(accountID, since)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load engagement"))
	}
	replies, err := s.DB.SelectPostRepliesForAccountSince(accountID, since)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load replies"))
	}

	sum := postengagement.SummariseInteractions(time.Now(), days, postengagement.InteractionSources{
		Posts:              posts,
		Replies:            replies,
		Events:             events,
		RepliesFromThreads: s.connectorSupportsReplies(sa.Connector),
	}, recentInteractionsMaxItems)

	res := &controlv1.GetSocialAccountRecentInteractionsResponse{
		Days:            uint32(days),
		Supported:       s.connectorSupportsEngagement(sa.Connector),
		TrackingEnabled: s.postEngagementMaxAge() > 0,
		Posts:           sum.Posts,
		Totals:          countsToProto(sum.Counts),
	}
	for _, b := range sum.Daily {
		res.Daily = append(res.Daily, &controlv1.PostPerformanceDay{
			Date:       b.Date,
			Posts:      b.Posts,
			Engagement: countsToProto(b.Counts),
		})
	}
	for _, it := range sum.Items {
		res.Items = append(res.Items, recentInteractionToProto(it))
	}

	return connect.NewResponse(res), nil
}
