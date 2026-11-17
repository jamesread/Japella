package mastodon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	req, err := newMastodonSearchRequest(ctx, sa, keyword, clampMastodonLimit(limit))
	if err != nil {
		return nil, err
	}
	parsed, err := doMastodonSearch(req)
	if err != nil {
		return nil, err
	}
	return filterSearchStatuses(parsed.Statuses, since), nil
}

func newMastodonSearchRequest(ctx context.Context, sa *connector.SocialAccount, keyword string, limit int) (*http.Request, error) {
	q := url.Values{}
	q.Set("q", keyword)
	q.Set("type", "statuses")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("resolve", "false")
	searchURL := mastodonInstance(sa) + "/api/v2/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sa.OAuthToken)
	return req, nil
}

func doMastodonSearch(req *http.Request) (mastodonSearchResponse, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return mastodonSearchResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if err := checkMastodonHTTP(resp, body, "mastodon search rate limited", "mastodon search returned %d: %s"); err != nil {
		return mastodonSearchResponse{}, err
	}
	var parsed mastodonSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return mastodonSearchResponse{}, fmt.Errorf("decode mastodon search: %w", err)
	}
	return parsed, nil
}

func filterSearchStatuses(statuses []TimelineStatus, since time.Time) []*connector.RemotePost {
	out := make([]*connector.RemotePost, 0, len(statuses))
	for _, st := range statuses {
		if rp := remotePostIfAfter(st, since); rp != nil {
			out = append(out, rp)
		}
	}
	return out
}

func remotePostIfAfter(st TimelineStatus, since time.Time) *connector.RemotePost {
	if !since.IsZero() && st.CreatedAt.Before(since) {
		return nil
	}
	return timelineStatusToRemotePost(st)
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
	runCtx, cancel := context.WithCancel(ctx)
	h := &mastodonWatchHandle{cancel: cancel}
	instanceURL := mastodonInstance(sa)
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
	for c.runHashtagStreamAttempt(ctx, instanceURL, token, keyword, tag, allKeywords, sink, &backoff) {
	}
}

func (c *MastodonConnector) runHashtagStreamAttempt(ctx context.Context, instanceURL, token, keyword, tag string, allKeywords []string, sink connector.MatchSink, backoff *time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	err := c.streamHashtagOnce(ctx, instanceURL, token, keyword, tag, allKeywords, sink)
	if ctx.Err() != nil {
		return false
	}
	if waitRateLimit(ctx, err, tag, c) {
		return true
	}
	if err != nil {
		c.Logger().Warnf("mastodon hashtag stream #%s: %v", tag, err)
	}
	return sleepBackoff(ctx, backoff)
}

func waitRateLimit(ctx context.Context, err error, tag string, c *MastodonConnector) bool {
	rl, ok := errors.AsType[*connector.RateLimitedError](err)
	if !ok {
		return false
	}
	wait := rl.RetryAfter
	if wait <= 0 {
		wait = 5 * time.Minute
	}
	c.Logger().Warnf("mastodon hashtag stream rate limited for #%s, waiting %s", tag, wait)
	select {
	case <-ctx.Done():
	case <-time.After(wait):
	}
	return true
}

func sleepBackoff(ctx context.Context, backoff *time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(*backoff):
	}
	if *backoff < 60*time.Second {
		*backoff *= 2
	}
	return true
}

func (c *MastodonConnector) streamHashtagOnce(ctx context.Context, instanceURL, token, keyword, tag string, allKeywords []string, sink connector.MatchSink) error {
	resp, err := openHashtagStream(ctx, instanceURL, token, tag)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := checkStreamResponse(resp); err != nil {
		return err
	}
	return readHashtagSSE(ctx, resp.Body, keyword, allKeywords, sink)
}

func openHashtagStream(ctx context.Context, instanceURL, token, tag string) (*http.Response, error) {
	streamURL := instanceURL + "/api/v1/streaming/hashtag?tag=" + url.QueryEscape(tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	return http.DefaultClient.Do(req)
}

func checkStreamResponse(resp *http.Response) error {
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
	return nil
}

func readHashtagSSE(ctx context.Context, body io.Reader, keyword string, allKeywords []string, sink connector.MatchSink) error {
	reader := bufio.NewReader(body)
	var eventName string
	var dataBuf strings.Builder
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
			flushHashtagEvent(eventName, dataBuf.String(), keyword, allKeywords, sink)
			eventName = ""
			dataBuf.Reset()
			continue
		}
		eventName = applySSELine(line, eventName, &dataBuf)
	}
}

func applySSELine(line, eventName string, dataBuf *strings.Builder) string {
	if after, ok := strings.CutPrefix(line, "event:"); ok {
		return strings.TrimSpace(after)
	}
	if strings.HasPrefix(line, "data:") {
		if dataBuf.Len() > 0 {
			dataBuf.WriteByte('\n')
		}
		dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
	}
	return eventName
}

func flushHashtagEvent(eventName, data, keyword string, allKeywords []string, sink connector.MatchSink) {
	data = strings.TrimSpace(data)
	if eventName != "update" || data == "" {
		return
	}
	post := parseHashtagUpdate(data)
	if post == nil {
		return
	}
	matched := keywordwatch.MatchKeywordsAgainstPost(post.Content, allKeywords)
	if len(matched) == 0 {
		matched = []string{keyword}
	}
	sink(post, matched)
}

func parseHashtagUpdate(data string) *connector.RemotePost {
	var status TimelineStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil
	}
	return timelineStatusToRemotePost(status)
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if d, ok := parseRetryAfterSeconds(v); ok {
		return d
	}
	return parseRetryAfterHTTPDate(v)
}

func parseRetryAfterSeconds(v string) (time.Duration, bool) {
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

func parseRetryAfterHTTPDate(v string) time.Duration {
	t, err := http.ParseTime(v)
	if err != nil {
		return 0
	}
	d := time.Until(t)
	if d > 0 {
		return d
	}
	return 0
}
