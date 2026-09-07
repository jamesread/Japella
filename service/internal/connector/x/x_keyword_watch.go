package x

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

type xSearchResponse struct {
	Data []struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		AuthorID  string `json:"author_id"`
		CreatedAt string `json:"created_at"`
	} `json:"data"`
	Includes struct {
		Users []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"users"`
	} `json:"includes"`
	Meta struct {
		ResultCount int `json:"result_count"`
	} `json:"meta"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (x *XConnector) PreferNativeWatch(sa *connector.SocialAccount, keywords []string) bool {
	// Filtered stream requires elevated product access; not enabled by default.
	return false
}

func (x *XConnector) NativeWatchFidelity(sa *connector.SocialAccount, keywords []string) string {
	return connector.NativeFidelitySearchOnly
}

func (x *XConnector) StartKeywordWatch(ctx context.Context, sa *connector.SocialAccount, keywords []string, sink connector.MatchSink) (connector.WatchHandle, error) {
	return nil, fmt.Errorf("x filtered stream native watch is not configured")
}

func (x *XConnector) SearchRecentByKeyword(ctx context.Context, sa *connector.SocialAccount, keyword string, since time.Time, limit int) ([]*connector.RemotePost, error) {
	if sa == nil || sa.OAuthToken == "" {
		return nil, fmt.Errorf("x social account token required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := url.Values{}
	q.Set("query", keyword)
	q.Set("max_results", strconv.Itoa(limit))
	q.Set("tweet.fields", "created_at,author_id")
	q.Set("expansions", "author_id")
	q.Set("user.fields", "username")
	if !since.IsZero() {
		q.Set("start_time", since.UTC().Format(time.RFC3339))
	}

	endpoint := "https://api.x.com/2/tweets/search/recent?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := 15 * time.Minute
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				retry = time.Duration(secs) * time.Second
			}
		}
		return nil, &connector.RateLimitedError{RetryAfter: retry, Message: "x search rate limited"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("x search returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed xSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	users := map[string]string{}
	for _, u := range parsed.Includes.Users {
		users[u.ID] = u.Username
	}

	out := make([]*connector.RemotePost, 0, len(parsed.Data))
	for _, tw := range parsed.Data {
		postedAt, _ := time.Parse(time.RFC3339, tw.CreatedAt)
		handle := users[tw.AuthorID]
		remoteURL := ""
		if handle != "" {
			remoteURL = fmt.Sprintf("https://x.com/%s/status/%s", handle, tw.ID)
		}
		out = append(out, &connector.RemotePost{
			RemoteID:     tw.ID,
			RemoteURL:    remoteURL,
			AuthorID:     tw.AuthorID,
			AuthorHandle: handle,
			Content:      tw.Text,
			PostedAt:     postedAt,
		})
	}
	return out, nil
}
