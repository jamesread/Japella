// Package feedcolumns validates feed column queries and caches the posts
// fetched for them.
package feedcolumns

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/posttags"
)

// MaxQueryLength matches feed_columns.query.
const MaxQueryLength = 255

var hashtagRe = regexp.MustCompile(`^#?([\p{L}\p{N}_]+)$`)

// Query is a parsed column query.
type Query struct {
	// Text is the query as stored: trimmed, with hashtags normalised to "#tag".
	Text string
	// Hashtag is the tag without '#', set when the query is a single hashtag.
	Hashtag string
	// Tag is set for "tag:<tag>" queries, which list the user's own posts
	// with that internal Japella tag rather than searching any network.
	Tag string
}

// TagQueryPrefix starts a query for an internal tag.
const TagQueryPrefix = "tag:"

// TagQuery returns the column query listing posts with an internal tag.
func TagQuery(tag string) string {
	return TagQueryPrefix + tag
}

func parseTagQuery(q string) (Query, bool, error) {
	if len(q) < len(TagQueryPrefix) || !strings.EqualFold(q[:len(TagQueryPrefix)], TagQueryPrefix) {
		return Query{}, false, nil
	}
	tag, err := posttags.Normalise(q[len(TagQueryPrefix):])
	if err != nil {
		return Query{}, true, err
	}
	return Query{Text: TagQuery(tag), Tag: tag}, true, nil
}

func parseHashtagQuery(q string) (Query, error) {
	m := hashtagRe.FindStringSubmatch(q)
	if m == nil {
		return Query{}, fmt.Errorf("hashtag may only contain letters, numbers and underscores")
	}
	return Query{Text: "#" + m[1], Hashtag: m[1]}, nil
}

// ParseQuery validates and normalises a column query. "tag:<tag>" is an
// internal tag, a single word starting with '#' is a hashtag, and anything
// else is a free-text search.
func ParseQuery(raw string) (Query, error) {
	q := strings.Join(strings.Fields(raw), " ")
	if q == "" {
		return Query{}, fmt.Errorf("query is required")
	}
	if parsed, ok, err := parseTagQuery(q); ok {
		return parsed, err
	}
	if utf8.RuneCountInString(q) > MaxQueryLength {
		return Query{}, fmt.Errorf("query must be at most %d characters", MaxQueryLength)
	}
	if strings.HasPrefix(q, "#") {
		return parseHashtagQuery(q)
	}
	return Query{Text: q}, nil
}

// DefaultTitle is used when a column has no title of its own.
func DefaultTitle(q Query) string {
	if q.Tag != "" {
		return q.Tag
	}
	return q.Text
}

// Result is one fetch of a column's posts.
type Result struct {
	Posts      []*connector.RemotePost
	FetchedAt  time.Time
	Err        error
	RetryAfter time.Time
}

type cacheEntry struct {
	key    string
	result Result
}

// Cache keeps the latest result per column and account. Fresh results are reused for TTL;
// after a failure the last good posts are kept alongside the error, and a
// rate limit is honoured until it expires.
type Cache struct {
	TTL time.Duration

	mu      sync.Mutex
	entries map[entryID]*cacheEntry
}

type entryID struct {
	column  uint32
	account uint32
}

// NewCache returns a cache that reuses results for ttl.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{TTL: ttl, entries: map[entryID]*cacheEntry{}}
}

func cacheFresh(r Result, ttl time.Duration, now time.Time) bool {
	if now.Before(r.RetryAfter) {
		return true
	}
	return r.Err == nil && now.Sub(r.FetchedAt) < ttl
}

func (c *Cache) cachedEntry(id entryID, key string) *cacheEntry {
	entry := c.entries[id]
	if entry != nil && entry.key != key {
		return nil
	}
	return entry
}

func resultFromFetch(posts []*connector.RemotePost, err error, now time.Time, entry *cacheEntry) Result {
	if err == nil {
		return Result{Posts: posts, FetchedAt: now}
	}
	result := Result{Err: err, FetchedAt: now}
	if entry != nil {
		result.Posts = entry.result.Posts
		result.FetchedAt = entry.result.FetchedAt
	}
	if rl := connector.AsRateLimited(err); rl != nil {
		result.RetryAfter = now.Add(rl.RetryAfter)
	}
	return result
}

// Get returns the result for one account of a column, calling fetch when the
// cached one is stale. key identifies what was fetched (the query) so an
// edited column is not served old posts.
func (c *Cache) Get(columnID, accountID uint32, key string, now time.Time, fetch func() ([]*connector.RemotePost, error)) Result {
	id := entryID{column: columnID, account: accountID}

	c.mu.Lock()
	entry := c.cachedEntry(id, key)
	if entry != nil && cacheFresh(entry.result, c.TTL, now) {
		r := entry.result
		c.mu.Unlock()
		return r
	}
	c.mu.Unlock()

	posts, err := fetch()

	c.mu.Lock()
	defer c.mu.Unlock()
	result := resultFromFetch(posts, err, now, entry)
	c.entries[id] = &cacheEntry{key: key, result: result}
	return result
}

// Forget drops a column's cached results for every account.
func (c *Cache) Forget(columnID uint32) {
	c.mu.Lock()
	for id := range c.entries {
		if id.column == columnID {
			delete(c.entries, id)
		}
	}
	c.mu.Unlock()
}

// SourcedPost is a post and the account it was found through.
type SourcedPost struct {
	Post      *connector.RemotePost
	AccountID uint32
}

// Merge combines results from several accounts: newest first, posts seen
// through more than one account kept once (the first account wins), capped
// at limit.
func Merge(byAccount map[uint32][]*connector.RemotePost, accountOrder []uint32, limit int) []SourcedPost {
	seen := map[string]bool{}
	out := make([]SourcedPost, 0)
	for _, accountID := range accountOrder {
		out = appendAccountPosts(out, seen, accountID, byAccount[accountID])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Post.PostedAt.After(out[j].Post.PostedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func appendAccountPosts(out []SourcedPost, seen map[string]bool, accountID uint32, posts []*connector.RemotePost) []SourcedPost {
	for _, p := range posts {
		if p == nil || alreadySeenRemotePost(seen, p) {
			continue
		}
		out = append(out, SourcedPost{Post: p, AccountID: accountID})
	}
	return out
}

func alreadySeenRemotePost(seen map[string]bool, p *connector.RemotePost) bool {
	key := p.RemoteURL
	if key == "" {
		key = p.RemoteID
	}
	if key == "" {
		return false
	}
	if seen[key] {
		return true
	}
	seen[key] = true
	return false
}
