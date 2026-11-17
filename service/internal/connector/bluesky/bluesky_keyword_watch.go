package bluesky

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jamesread/japella/internal/connector"
)

type bskySearchPostsResponse struct {
	Posts []bskySearchPost `json:"posts"`
}

type bskySearchPost struct {
	URI    string `json:"uri"`
	Author struct {
		Did         string `json:"did"`
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
		Avatar      string `json:"avatar"`
	} `json:"author"`
	Record struct {
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"createdAt"`
		Langs     []string  `json:"langs"`
	} `json:"record"`
}

func (c *BlueskyConnector) PreferNativeWatch(sa *connector.SocialAccount, keywords []string) bool {
	// Jetstream has no server-side keyword filter; use search polling.
	return false
}

func (c *BlueskyConnector) NativeWatchFidelity(sa *connector.SocialAccount, keywords []string) string {
	return connector.NativeFidelitySearchOnly
}

func (c *BlueskyConnector) StartKeywordWatch(ctx context.Context, sa *connector.SocialAccount, keywords []string, sink connector.MatchSink) (connector.WatchHandle, error) {
	return nil, fmt.Errorf("bluesky native keyword watch is not available; use search mode")
}

func (c *BlueskyConnector) SearchRecentByKeyword(ctx context.Context, sa *connector.SocialAccount, keyword string, since time.Time, limit int) ([]*connector.RemotePost, error) {
	limit = clampBlueskySearchLimit(limit)
	req, err := newBlueskySearchRequest(ctx, sa, keyword, limit)
	if err != nil {
		return nil, err
	}
	parsed, err := doBlueskySearch(req)
	if err != nil {
		return nil, err
	}
	return mapBlueskySearchPosts(parsed, since), nil
}

func clampBlueskySearchLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func newBlueskySearchRequest(ctx context.Context, sa *connector.SocialAccount, keyword string, limit int) (*http.Request, error) {
	q := url.Values{}
	q.Set("q", keyword)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sort", "latest")
	endpoint := "https://api.bsky.app/xrpc/app.bsky.feed.searchPosts?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if sa != nil && sa.OAuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)
	}
	return req, nil
}

func doBlueskySearch(req *http.Request) (bskySearchPostsResponse, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return bskySearchPostsResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if err := checkBlueskySearchStatus(resp.StatusCode, body); err != nil {
		return bskySearchPostsResponse{}, err
	}
	var parsed bskySearchPostsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return bskySearchPostsResponse{}, err
	}
	return parsed, nil
}

func checkBlueskySearchStatus(statusCode int, body []byte) error {
	if statusCode == http.StatusTooManyRequests {
		return &connector.RateLimitedError{
			RetryAfter: 15 * time.Minute,
			Message:    "bluesky search rate limited",
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("bluesky search returned %d: %s", statusCode, string(body))
	}
	return nil
}

func mapBlueskySearchPosts(parsed bskySearchPostsResponse, since time.Time) []*connector.RemotePost {
	out := make([]*connector.RemotePost, 0, len(parsed.Posts))
	for _, p := range parsed.Posts {
		if shouldSkipBlueskySearchPost(p, since) {
			continue
		}
		out = append(out, blueskySearchPostToRemote(p))
	}
	return out
}

func shouldSkipBlueskySearchPost(p bskySearchPost, since time.Time) bool {
	return !since.IsZero() && !p.Record.CreatedAt.IsZero() && p.Record.CreatedAt.Before(since)
}

func blueskySearchPostToRemote(p bskySearchPost) *connector.RemotePost {
	return &connector.RemotePost{
		RemoteID:     p.URI,
		RemoteURL:    blueskyWebURL(p.URI, p.Author.Handle),
		AuthorID:     p.Author.Did,
		AuthorHandle: p.Author.Handle,
		Content:      p.Record.Text,
		PostedAt:     p.Record.CreatedAt,

		AuthorName:       p.Author.DisplayName,
		AuthorAvatarURL:  p.Author.Avatar,
		AuthorProfileURL: blueskyProfileURL(p.Author.Handle, p.Author.Did),
		Language:         firstOrEmpty(p.Record.Langs),
	}
}

// firstOrEmpty returns the first language a post declares; Bluesky lists the
// author's choice first.
func firstOrEmpty(langs []string) string {
	if len(langs) == 0 {
		return ""
	}
	return langs[0]
}

func blueskyProfileURL(handle, did string) string {
	id := handle
	if id == "" {
		id = did
	}
	if id == "" {
		return ""
	}
	return "https://bsky.app/profile/" + url.PathEscape(id)
}

func blueskyWebURL(atURI, handle string) string {
	parts := strings.Split(atURI, "/")
	if len(parts) < 5 || handle == "" {
		return ""
	}
	rkey := parts[len(parts)-1]
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, rkey)
}
