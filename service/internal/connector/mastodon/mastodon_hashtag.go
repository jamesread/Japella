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
	if limit <= 0 {
		limit = 20
	}
	if limit > 40 {
		limit = 40
	}

	instanceURL := sa.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	endpoint := instanceURL + "/api/v1/timelines/tag/" + url.PathEscape(tag) + "?limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &connector.RateLimitedError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")), Message: "mastodon tag timeline rate limited"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mastodon tag timeline returned %d: %s", resp.StatusCode, string(body))
	}

	var statuses []TimelineStatus
	if err := json.Unmarshal(body, &statuses); err != nil {
		return nil, fmt.Errorf("decode mastodon tag timeline: %w", err)
	}

	out := make([]*connector.RemotePost, 0, len(statuses))
	for _, st := range statuses {
		if rp := timelineStatusToRemotePost(st); rp != nil {
			out = append(out, rp)
		}
	}
	return out, nil
}
