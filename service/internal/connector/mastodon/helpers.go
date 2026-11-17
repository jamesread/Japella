package mastodon

import (
	"fmt"
	"net/http"

	"github.com/jamesread/japella/internal/connector"
)

func mastodonInstance(sa *connector.SocialAccount) string {
	if sa.Homeserver == "" {
		return defaultMastodonInstance
	}
	return sa.Homeserver
}

func clampMastodonLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 40 {
		return 40
	}
	return limit
}

func checkMastodonHTTP(resp *http.Response, body []byte, rateMsg, errFmt string) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		return &connector.RateLimitedError{
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
			Message:    rateMsg,
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(errFmt, resp.StatusCode, string(body))
	}
	return nil
}

func statusesToRemotePosts(statuses []TimelineStatus) []*connector.RemotePost {
	out := make([]*connector.RemotePost, 0, len(statuses))
	for _, st := range statuses {
		if rp := timelineStatusToRemotePost(st); rp != nil {
			out = append(out, rp)
		}
	}
	return out
}
