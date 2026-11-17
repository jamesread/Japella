package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jamesread/japella/internal/connector"
)

// FetchHashtagTimeline lists recent statuses with a hashtag from the account's
// instance. Unlike /api/v2/search, this works without full-text search enabled.
func (c *MastodonConnector) FetchHashtagTimeline(ctx context.Context, sa *connector.SocialAccount, tag string, limit int) ([]*connector.RemotePost, error) {
	if sa == nil {
		return nil, fmt.Errorf("social account required")
	}
	req, err := newHashtagTimelineRequest(ctx, sa, tag, clampMastodonLimit(limit))
	if err != nil {
		return nil, err
	}
	statuses, err := doMastodonTimelineGET(req, "mastodon tag timeline")
	if err != nil {
		return nil, err
	}
	return statusesToRemotePosts(statuses), nil
}

func newHashtagTimelineRequest(ctx context.Context, sa *connector.SocialAccount, tag string, limit int) (*http.Request, error) {
	endpoint := mastodonInstance(sa) + "/api/v1/timelines/tag/" + url.PathEscape(tag) + "?limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)
	return req, nil
}

func doMastodonTimelineGET(req *http.Request, rateLimitMsg string) ([]TimelineStatus, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if err := checkMastodonHTTP(resp, body, rateLimitMsg+" rate limited", rateLimitMsg+" returned %d: %s"); err != nil {
		return nil, err
	}
	var statuses []TimelineStatus
	if err := json.Unmarshal(body, &statuses); err != nil {
		return nil, fmt.Errorf("decode %s: %w", rateLimitMsg, err)
	}
	return statuses, nil
}
