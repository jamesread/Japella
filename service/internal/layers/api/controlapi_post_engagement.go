package api

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/postengagement"
	"github.com/jamesread/japella/internal/postreplies"
	"github.com/jamesread/japella/internal/webhook"
	log "github.com/sirupsen/logrus"
)

const postEngagementJobName = "post_engagement"

// postEngagementEventRetention is how long detected engagement increases are
// kept for the recent interactions view.
const postEngagementEventRetention = 90 * 24 * time.Hour

func postEngagementLog(sa *db.SocialAccount) *log.Entry {
	fields := log.Fields{"component": "PostEngagement"}
	if sa != nil {
		fields["connector"] = sa.Connector
		fields["social_account_id"] = sa.ID
	}
	return log.WithFields(fields)
}

func (s *ControlApi) postEngagementInterval() time.Duration {
	secs := 900
	if s.DB != nil {
		if v := int(s.DB.GetCvarInt(db.CvarKeys.PostEngagementIntervalSeconds)); v > 0 {
			secs = v
		}
	}
	return time.Duration(secs) * time.Second
}

func (s *ControlApi) postEngagementMaxAge() time.Duration {
	if s.DB == nil {
		return 0
	}
	days := int(s.DB.GetCvarInt(db.CvarKeys.PostEngagementMaxAgeDays))
	if days <= 0 {
		return 0
	}
	return time.Duration(days) * 24 * time.Hour
}

func (s *ControlApi) postEngagementMaxPosts() int {
	n := 100
	if s.DB != nil {
		if v := int(s.DB.GetCvarInt(db.CvarKeys.PostEngagementMaxPostsPerRun)); v > 0 {
			n = v
		}
	}
	return n
}

// startPostEngagementJob runs processPostEngagement on a cvar-controlled interval.
func (s *ControlApi) startPostEngagementJob(recordJobRun func(string)) {
	for s.cc == nil {
		time.Sleep(200 * time.Millisecond)
	}

	interval := s.postEngagementInterval()
	recordJobRun(postEngagementJobName)
	s.processPostEngagement()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		<-ticker.C
		if next := s.postEngagementInterval(); next != interval {
			ticker.Reset(next)
			interval = next
		}
		recordJobRun(postEngagementJobName)
		s.processPostEngagement()
	}
}

// processPostEngagement fetches engagement counters for recently published
// posts, stores them, and emits post.engagement when there is new activity.
func (s *ControlApi) processPostEngagement() {
	if s.cc == nil || s.DB == nil || !s.DB.Ready() {
		return
	}

	maxAge := s.postEngagementMaxAge()
	if maxAge == 0 {
		return
	}

	_ = s.DB.DeletePostEngagementEventsBefore(time.Now().UTC().Add(-postEngagementEventRetention))

	posts, err := s.DB.SelectPostsForEngagementFetch(maxAge, s.postEngagementMaxPosts())
	if err != nil || len(posts) == 0 {
		return
	}

	accounts := map[uint32]*db.SocialAccount{}
	rateLimited := map[uint32]bool{}
	fetched, notified, failed, threads := 0, 0, 0, 0

	for _, post := range posts {
		saID := post.SocialAccountIDUint()
		if rateLimited[saID] {
			continue
		}

		sa, ok := accounts[saID]
		if !ok {
			sa, err = s.DB.GetSocialAccount(saID)
			if err != nil || sa == nil {
				continue
			}
			accounts[saID] = sa
		}

		engagementConnector, svc := s.postEngagementConnector(sa)
		if engagementConnector == nil {
			continue
		}

		if socialAccountNeedsTokenRefresh(sa, time.Hour) {
			if refreshed, refreshErr := s.refreshSocialAccountToken(sa, svc); refreshErr == nil {
				sa = refreshed
				accounts[saID] = sa
			}
		}

		result, fetchErr := s.fetchPostEngagement(engagementConnector, sa, post)
		if rl := connector.AsRateLimited(fetchErr); rl != nil {
			postEngagementLog(sa).Warnf("rate limited, skipping remaining posts this run (retry after %s)", rl.RetryAfter)
			rateLimited[saID] = true
			continue
		}
		if fetchErr != nil {
			failed++
			postEngagementLog(sa).WithField("post_id", post.ID).Warnf("failed to fetch engagement: %v", fetchErr)
			_ = s.DB.MarkPostEngagementError(post.ID, fetchErr.Error())
			continue
		}

		fetched++
		stored, dispatched := s.recordPostEngagement(post, sa, result)
		if dispatched {
			notified++
		}

		synced, repliesErr := s.syncPostReplies(sa, post, stored, result.Replies)
		if rl := connector.AsRateLimited(repliesErr); rl != nil {
			postEngagementLog(sa).Warnf("rate limited fetching replies, skipping remaining posts this run (retry after %s)", rl.RetryAfter)
			rateLimited[saID] = true
			continue
		}
		if repliesErr != nil {
			postEngagementLog(sa).WithField("post_id", post.ID).Warnf("failed to fetch replies: %v", repliesErr)
		}
		if synced {
			threads++
		}
	}

	log.WithField("component", "PostEngagement").Infof("run complete: fetched %d, notified %d, failed %d, reply threads updated %d", fetched, notified, failed, threads)
}

// syncPostReplies refreshes the stored reply thread when the replies counter
// says it is out of date. stored is the engagement row from before this run.
// It returns true when replies were fetched and stored.
func (s *ControlApi) syncPostReplies(sa *db.SocialAccount, post *db.Post, stored *db.PostEngagement, replyCount uint32) (synced bool, err error) {
	rc, ok := s.cc.Get(sa.Connector).(connector.ConnectorWithPostReplies)
	if !ok || !postreplies.ShouldFetch(stored, replyCount) {
		return false, nil
	}

	defer func() {
		if r := recover(); r != nil {
			synced, err = false, fmt.Errorf("panic while fetching replies: %v", r)
		}
	}()

	fetched, err := rc.FetchPostReplies(toConnectorSA(sa), post.RemoteID.String, post.PostURL.String)
	if err != nil {
		return false, err
	}

	if err := s.DB.ReplacePostReplies(post.ID, postreplies.ToDB(post.ID, fetched), replyCount); err != nil {
		return false, err
	}
	return true, nil
}

func (s *ControlApi) postEngagementConnector(sa *db.SocialAccount) (connector.ConnectorWithPostEngagement, connector.BaseConnector) {
	svc := s.cc.Get(sa.Connector)
	if svc == nil {
		return nil, nil
	}
	ec, ok := svc.(connector.ConnectorWithPostEngagement)
	if !ok {
		return nil, nil
	}
	return ec, svc
}

func (s *ControlApi) fetchPostEngagement(ec connector.ConnectorWithPostEngagement, sa *db.SocialAccount, post *db.Post) (result *connector.PostEngagement, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while fetching engagement: %v", r)
		}
	}()
	return ec.FetchPostEngagement(toConnectorSA(sa), post.RemoteID.String, post.PostURL.String)
}

// recordPostEngagement stores the snapshot and dispatches a webhook when there
// is new activity. It returns the row as it was before this snapshot (nil if
// none) and whether a notification was dispatched.
func (s *ControlApi) recordPostEngagement(post *db.Post, sa *db.SocialAccount, result *connector.PostEngagement) (*db.PostEngagement, bool) {
	stored, err := s.DB.GetPostEngagement(post.ID)
	if err != nil {
		return nil, false
	}

	cmp := postengagement.Compare(stored, postengagement.FromConnector(result))
	if err := s.DB.UpsertPostEngagement(postengagement.ToDB(post.ID, cmp.Current), cmp.Changed); err != nil {
		return stored, false
	}

	if !cmp.HasNewActivity() {
		return stored, false
	}

	_ = s.DB.InsertPostEngagementEvent(&db.PostEngagementEvent{
		PostID:          post.ID,
		SocialAccountID: post.SocialAccountIDUint(),
		Likes:           cmp.Delta.Likes,
		Reposts:         cmp.Delta.Reposts,
		Replies:         cmp.Delta.Replies,
		Quotes:          cmp.Delta.Quotes,
	})

	s.webhookDispatcher().Dispatch(context.Background(), webhook.EventPostEngagement, s.buildPostEngagementWebhookPayload(post, sa, cmp))
	return stored, true
}

func (s *ControlApi) buildPostEngagementWebhookPayload(post *db.Post, sa *db.SocialAccount, cmp postengagement.Comparison) map[string]any {
	payload := s.buildPostWebhookPayload(post, sa, db.PostStateCompleted, post.PostURL.String)
	payload["engagement"] = map[string]any{
		"current":  cmp.Current,
		"previous": cmp.Previous,
		"delta":    cmp.Delta,
	}
	return payload
}

func (s *ControlApi) postEngagementSchedule() string {
	if s.postEngagementMaxAge() == 0 {
		return "Disabled (cvar)"
	}
	return fmt.Sprintf("Every %s (cvar)", s.postEngagementInterval())
}

const (
	postPerformanceDefaultDays = 7
	postPerformanceMaxDays     = 90
	postPerformanceMaxPosts    = 200
)

// engagementToProto converts a stored row; nil means never fetched.
func engagementToProto(pe *db.PostEngagement) *controlv1.PostEngagement {
	if pe == nil {
		return nil
	}
	out := countsToProto(postengagement.FromDB(pe))
	out.LastError = pe.LastError
	out.HasData = postengagement.HasBaseline(pe)
	if pe.LastFetchedAt.Valid {
		out.LastFetchedAt = pe.LastFetchedAt.Time.UTC().Format(time.RFC3339)
	}
	return out
}

func countsToProto(c postengagement.Counts) *controlv1.PostEngagement {
	return &controlv1.PostEngagement{
		Likes:       c.Likes,
		Reposts:     c.Reposts,
		Replies:     c.Replies,
		Quotes:      c.Quotes,
		Impressions: c.Impressions,
	}
}

// attachEngagement fills PostStatus.Engagement for a batch of statuses in one query.
func (s *ControlApi) attachEngagement(statuses []*controlv1.PostStatus) {
	if s.DB == nil || len(statuses) == 0 {
		return
	}
	ids := make([]uint32, 0, len(statuses))
	for _, st := range statuses {
		if st != nil && st.Id != 0 {
			ids = append(ids, st.Id)
		}
	}
	rows, err := s.DB.SelectPostEngagementForPosts(ids)
	if err != nil {
		return
	}
	for _, st := range statuses {
		if st == nil {
			continue
		}
		st.Engagement = engagementToProto(rows[st.Id])
	}
}

func (s *ControlApi) connectorSupportsEngagement(connectorName string) bool {
	if s.cc == nil {
		return false
	}
	svc := s.cc.Get(connectorName)
	if svc == nil {
		return false
	}
	_, ok := svc.(connector.ConnectorWithPostEngagement)
	return ok
}

func clampPerformanceDays(days uint32) int {
	if days == 0 {
		return postPerformanceDefaultDays
	}
	if days > postPerformanceMaxDays {
		return postPerformanceMaxDays
	}
	return int(days)
}

func postedAt(p *db.Post) time.Time {
	if p.ScheduledAt.Valid {
		return p.ScheduledAt.Time
	}
	return p.CreatedAt
}

// GetSocialAccountPostPerformance summarises engagement on an account's posts
// published during the last N calendar days.
func (s *ControlApi) GetSocialAccountPostPerformance(ctx context.Context, req *connect.Request[controlv1.GetSocialAccountPostPerformanceRequest]) (*connect.Response[controlv1.GetSocialAccountPostPerformanceResponse], error) {
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

	days := clampPerformanceDays(req.Msg.GetDays())

	posts, err := s.DB.SelectCompletedPostsForAccountSince(accountID, days)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load posts"))
	}

	ids := make([]uint32, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	rows, err := s.DB.SelectPostEngagementForPosts(ids)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load engagement"))
	}

	byID := make(map[uint32]*db.Post, len(posts))
	samples := make([]postengagement.PostSample, len(posts))
	for i, p := range posts {
		byID[p.ID] = p
		samples[i] = postengagement.PostSample{
			PostID:   p.ID,
			PostedAt: postedAt(p),
			Counts:   postengagement.FromDB(rows[p.ID]),
		}
	}

	res := &controlv1.GetSocialAccountPostPerformanceResponse{
		Days:            uint32(days),
		Supported:       s.connectorSupportsEngagement(sa.Connector),
		TrackingEnabled: s.postEngagementMaxAge() > 0,
		PostCount:       uint32(len(posts)),
		Totals:          countsToProto(postengagement.Total(samples)),
	}

	for _, b := range postengagement.Daily(time.Now(), days, samples) {
		res.Daily = append(res.Daily, &controlv1.PostPerformanceDay{
			Date:       b.Date,
			Posts:      b.Posts,
			Engagement: countsToProto(b.Counts),
		})
	}

	icon := "mdi:question-mark-circle"
	if s.cc != nil {
		if svc := s.cc.Get(sa.Connector); svc != nil {
			icon = svc.GetIcon()
		}
	}

	for _, sample := range postengagement.SortByInteractions(samples) {
		if len(res.Posts) >= postPerformanceMaxPosts {
			break
		}
		p := byID[sample.PostID]
		res.Posts = append(res.Posts, &controlv1.PostStatus{
			Id:                    p.ID,
			Created:               p.CreatedAt.Format("2006-01-02 15:04:05"),
			SocialAccountId:       accountID,
			SocialAccountIcon:     icon,
			SocialAccountIdentity: sa.Identity,
			Content:               p.Content,
			Success:               p.Status,
			PostUrl:               p.PostURL.String,
			CampaignId:            uint32(p.CampaignID.Int32),
			CampaignName:          p.CampaignName.String,
			State:                 p.State,
			PostedDate:            postedAt(p).Format("2006-01-02 15:04:05"),
			Engagement:            engagementToProto(rows[p.ID]),
		})
	}

	return connect.NewResponse(res), nil
}
