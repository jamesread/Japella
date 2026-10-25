package api

import (
	"context"
	"fmt"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/postengagement"
	"github.com/jamesread/japella/internal/webhook"
	log "github.com/sirupsen/logrus"
)

const postEngagementJobName = "post_engagement"

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

	posts, err := s.DB.SelectPostsForEngagementFetch(maxAge, s.postEngagementMaxPosts())
	if err != nil || len(posts) == 0 {
		return
	}

	accounts := map[uint32]*db.SocialAccount{}
	rateLimited := map[uint32]bool{}
	fetched, notified, failed := 0, 0, 0

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
		if s.recordPostEngagement(post, sa, result) {
			notified++
		}
	}

	log.WithField("component", "PostEngagement").Infof("run complete: fetched %d, notified %d, failed %d", fetched, notified, failed)
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
// is new activity. It returns true when a notification was dispatched.
func (s *ControlApi) recordPostEngagement(post *db.Post, sa *db.SocialAccount, result *connector.PostEngagement) bool {
	stored, err := s.DB.GetPostEngagement(post.ID)
	if err != nil {
		return false
	}

	cmp := postengagement.Compare(stored, postengagement.FromConnector(result))
	if err := s.DB.UpsertPostEngagement(postengagement.ToDB(post.ID, cmp.Current), cmp.Changed); err != nil {
		return false
	}

	if !cmp.HasNewActivity() {
		return false
	}

	s.webhookDispatcher().Dispatch(context.Background(), webhook.EventPostEngagement, s.buildPostEngagementWebhookPayload(post, sa, cmp))
	return true
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
