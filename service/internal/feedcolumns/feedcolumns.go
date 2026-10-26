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

// ParseQuery validates and normalises a column query. "tag:<tag>" is an
// internal tag, a single word starting with '#' is a hashtag, and anything
// else is a free-text search.
func ParseQuery(raw string) (Query, error) {
	q := strings.Join(strings.Fields(raw), " ")
	if q == "" {
		return Query{}, fmt.Errorf("query is required")
	}
	if len(q) >= len(TagQueryPrefix) && strings.EqualFold(q[:len(TagQueryPrefix)], TagQueryPrefix) {
		tag, err := posttags.Normalise(q[len(TagQueryPrefix):])
		if err != nil {
			return Query{}, err
		}
		return Query{Text: TagQuery(tag), Tag: tag}, nil
	}
	if utf8.RuneCountInString(q) > MaxQueryLength {
		return Query{}, fmt.Errorf("query must be at most %d characters", MaxQueryLength)
	}
	if strings.HasPrefix(q, "#") {
		m := hashtagRe.FindStringSubmatch(q)
		if m == nil {
			return Query{}, fmt.Errorf("hashtag may only contain letters, numbers and underscores")
		}
		return Query{Text: "#" + m[1], Hashtag: m[1]}, nil
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

// Get returns the result for one account of a column, calling fetch when the
// cached one is stale. key identifies what was fetched (the query) so an
// edited column is not served old posts.
func (c *Cache) Get(columnID, accountID uint32, key string, now time.Time, fetch func() ([]*connector.RemotePost, error)) Result {
	id := entryID{column: columnID, account: accountID}

	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil && entry.key != key {
		entry = nil
	}
	if entry != nil {
		r := entry.result
		if now.Before(r.RetryAfter) || (r.Err == nil && now.Sub(r.FetchedAt) < c.TTL) {
			c.mu.Unlock()
			return r
		}
	}
	c.mu.Unlock()

	posts, err := fetch()

	c.mu.Lock()
	defer c.mu.Unlock()

	result := Result{Posts: posts, FetchedAt: now}
	if err != nil {
		result = Result{Err: err, FetchedAt: now}
		if entry != nil {
			result.Posts = entry.result.Posts
			result.FetchedAt = entry.result.FetchedAt
		}
		if rl := connector.AsRateLimited(err); rl != nil {
			result.RetryAfter = now.Add(rl.RetryAfter)
		}
	}
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
		for _, p := range byAccount[accountID] {
			if p == nil {
				continue
			}
			key := p.RemoteURL
			if key == "" {
				key = p.RemoteID
			}
			if key != "" {
				if seen[key] {
					continue
				}
				seen[key] = true
			}
			out = append(out, SourcedPost{Post: p, AccountID: accountID})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Post.PostedAt.After(out[j].Post.PostedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
