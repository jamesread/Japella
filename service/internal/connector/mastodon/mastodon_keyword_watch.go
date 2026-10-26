package mastodon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/keywordwatch"
)

type mastodonSearchResponse struct {
	Statuses []TimelineStatus `json:"statuses"`
}

func (c *MastodonConnector) PreferNativeWatch(sa *connector.SocialAccount, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	for _, kw := range keywords {
		if !keywordwatch.IsHashtagKeyword(kw) {
			return false
		}
	}
	return true
}

func (c *MastodonConnector) NativeWatchFidelity(sa *connector.SocialAccount, keywords []string) string {
	if c.PreferNativeWatch(sa, keywords) {
		return connector.NativeFidelityHashtagOnly
	}
	return connector.NativeFidelitySearchOnly
}

func (c *MastodonConnector) SearchRecentByKeyword(ctx context.Context, sa *connector.SocialAccount, keyword string, since time.Time, limit int) ([]*connector.RemotePost, error) {
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

	q := url.Values{}
	q.Set("q", keyword)
	q.Set("type", "statuses")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("resolve", "false")

	searchURL := instanceURL + "/api/v2/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &connector.RateLimitedError{RetryAfter: retry, Message: "mastodon search rate limited"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mastodon search returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed mastodonSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode mastodon search: %w", err)
	}

	out := make([]*connector.RemotePost, 0, len(parsed.Statuses))
	for _, st := range parsed.Statuses {
		if !since.IsZero() && st.CreatedAt.Before(since) {
			continue
		}
		rp := timelineStatusToRemotePost(st)
		if rp != nil {
			out = append(out, rp)
		}
	}
	return out, nil
}

func timelineStatusToRemotePost(status TimelineStatus) *connector.RemotePost {
	feedPost, ok := timelineStatusToFeedPost(status)
	if !ok {
		return nil
	}
	return &connector.RemotePost{
		RemoteID:     feedPost.RemoteID,
		RemoteURL:    feedPost.RemoteURL,
		AuthorID:     feedPost.AuthorID,
		AuthorHandle: feedPost.AuthorName, // keyword.matched webhooks expose this; keep its format
		Content:      feedPost.Content,
		PostedAt:     feedPost.PostedDate,

		AuthorName:       feedPost.AuthorName,
		AuthorAvatarURL:  feedPost.AuthorAvatarURL,
		AuthorProfileURL: feedPost.AuthorProfileURL,
		ContentIsHTML:    true,
		Language:         feedPost.Language,
		PostInteraction:  feedPost.PostInteraction,
	}
}

type mastodonWatchHandle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (h *mastodonWatchHandle) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
}

func (c *MastodonConnector) StartKeywordWatch(ctx context.Context, sa *connector.SocialAccount, keywords []string, sink connector.MatchSink) (connector.WatchHandle, error) {
	if sink == nil {
		return nil, fmt.Errorf("match sink required")
	}
	if !c.PreferNativeWatch(sa, keywords) {
		return nil, fmt.Errorf("mastodon native watch requires hashtag-only keywords")
	}

	instanceURL := sa.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	runCtx, cancel := context.WithCancel(ctx)
	h := &mastodonWatchHandle{cancel: cancel}

	for _, kw := range keywords {
		tag := keywordwatch.HashtagTag(kw)
		h.wg.Add(1)
		go func(keyword, tag string) {
			defer h.wg.Done()
			c.streamHashtag(runCtx, instanceURL, sa.OAuthToken, keyword, tag, keywords, sink)
		}(kw, tag)
	}

	return h, nil
}

func (c *MastodonConnector) streamHashtag(ctx context.Context, instanceURL, token, keyword, tag string, allKeywords []string, sink connector.MatchSink) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := c.streamHashtagOnce(ctx, instanceURL, token, keyword, tag, allKeywords, sink)
		if ctx.Err() != nil {
			return
		}
		if rl, ok := err.(*connector.RateLimitedError); ok {
			wait := rl.RetryAfter
			if wait <= 0 {
				wait = 5 * time.Minute
			}
			c.Logger().Warnf("mastodon hashtag stream rate limited for #%s, waiting %s", tag, wait)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			continue
		}
		if err != nil {
			c.Logger().Warnf("mastodon hashtag stream #%s: %v", tag, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 60*time.Second {
			backoff *= 2
		}
	}
}

func (c *MastodonConnector) streamHashtagOnce(ctx context.Context, instanceURL, token, keyword, tag string, allKeywords []string, sink connector.MatchSink) error {
	streamURL := instanceURL + "/api/v1/streaming/hashtag?tag=" + url.QueryEscape(tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &connector.RateLimitedError{
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
			Message:    "mastodon streaming rate limited",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("streaming returned %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var eventName string
	var dataBuf strings.Builder

	flush := func() {
		data := strings.TrimSpace(dataBuf.String())
		dataBuf.Reset()
		name := eventName
		eventName = ""
		if name != "update" || data == "" {
			return
		}
		var status TimelineStatus
		if err := json.Unmarshal([]byte(data), &status); err != nil {
			return
		}
		post := timelineStatusToRemotePost(status)
		if post == nil {
			return
		}
		matched := keywordwatch.MatchKeywordsAgainstPost(post.Content, allKeywords)
		if len(matched) == 0 {
			matched = []string{keyword}
		}
		sink(post, matched)
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
