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
		Lang      string `json:"lang"`
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
	token, err := requireXOAuthToken(sa)
	if err != nil {
		return nil, err
	}
	req, err := x.newKeywordSearchRequest(ctx, token, keyword, since, clampXSearchLimit(limit))
	if err != nil {
		return nil, err
	}
	parsed, err := doXKeywordSearch(req)
	if err != nil {
		return nil, err
	}
	return mapXSearchToRemotePosts(parsed), nil
}

func requireXOAuthToken(sa *connector.SocialAccount) (string, error) {
	if sa == nil || sa.OAuthToken == "" {
		return "", fmt.Errorf("x social account token required")
	}
	return sa.OAuthToken, nil
}

func doXKeywordSearch(req *http.Request) (xSearchResponse, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return xSearchResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if err := checkXSearchHTTPStatus(resp, body); err != nil {
		return xSearchResponse{}, err
	}
	var parsed xSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return xSearchResponse{}, err
	}
	return parsed, nil
}

func clampXSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (x *XConnector) newKeywordSearchRequest(ctx context.Context, token, keyword string, since time.Time, limit int) (*http.Request, error) {
	q := url.Values{}
	q.Set("query", keyword)
	q.Set("max_results", strconv.Itoa(limit))
	q.Set("tweet.fields", "created_at,author_id,lang")
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
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func checkXSearchHTTPStatus(resp *http.Response, body []byte) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		return &connector.RateLimitedError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")), Message: "x search rate limited"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("x search returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func parseRetryAfter(ra string) time.Duration {
	retry := 15 * time.Minute
	if ra == "" {
		return retry
	}
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return retry
}

func mapXSearchToRemotePosts(parsed xSearchResponse) []*connector.RemotePost {
	users := map[string]string{}
	for _, u := range parsed.Includes.Users {
		users[u.ID] = u.Username
	}
	out := make([]*connector.RemotePost, 0, len(parsed.Data))
	for _, tw := range parsed.Data {
		out = append(out, xSearchTweetToRemote(tw, users[tw.AuthorID]))
	}
	return out
}

func xSearchTweetToRemote(tw struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	AuthorID  string `json:"author_id"`
	CreatedAt string `json:"created_at"`
	Lang      string `json:"lang"`
}, handle string) *connector.RemotePost {
	postedAt, _ := time.Parse(time.RFC3339, tw.CreatedAt)
	remoteURL, profileURL := "", ""
	if handle != "" {
		remoteURL = fmt.Sprintf("https://x.com/%s/status/%s", handle, tw.ID)
		profileURL = "https://x.com/" + handle
	}
	return &connector.RemotePost{
		RemoteID:         tw.ID,
		RemoteURL:        remoteURL,
		AuthorID:         tw.AuthorID,
		AuthorHandle:     handle,
		Content:          tw.Text,
		PostedAt:         postedAt,
		Language:         tw.Lang,
		AuthorProfileURL: profileURL,
	}
}
