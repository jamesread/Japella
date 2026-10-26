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
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &connector.RateLimitedError{
			RetryAfter: 15 * time.Minute,
			Message:    "bluesky search rate limited",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bluesky search returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed bskySearchPostsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	out := make([]*connector.RemotePost, 0, len(parsed.Posts))
	for _, p := range parsed.Posts {
		if !since.IsZero() && !p.Record.CreatedAt.IsZero() && p.Record.CreatedAt.Before(since) {
			continue
		}
		out = append(out, &connector.RemotePost{
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
		})
	}
	return out, nil
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
