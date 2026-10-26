package feedcolumns

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jamesread/japella/internal/connector"
)

func TestParseQuery(t *testing.T) {
	cases := []struct {
		in, text, tag string
		ok            bool
	}{
		{"#golang", "#golang", "golang", true},
		{"  #Café_2026 ", "#Café_2026", "Café_2026", true},
		{"japella   release", "japella release", "", true},
		{"#two words", "", "", false},
		{"#", "", "", false},
		{"   ", "", "", false},
		{strings.Repeat("a", 256), "", "", false},
		{"tag:x y", "", "", false},
		{"tag:", "", "", false},
	}
	for _, c := range cases {
		q, err := ParseQuery(c.in)
		if (err == nil) != c.ok {
			t.Fatalf("%q: ok=%v err=%v", c.in, c.ok, err)
		}
		if c.ok && (q.Text != c.text || q.Hashtag != c.tag) {
			t.Fatalf("%q: got %+v", c.in, q)
		}
	}
}

func TestParseTagQuery(t *testing.T) {
	for _, in := range []string{"tag:todo", "Tag:#ToDo", "  tag: todo "} {
		q, err := ParseQuery(in)
		if err != nil || q.Text != "tag:todo" || q.Tag != "todo" || q.Hashtag != "" {
			t.Fatalf("%q: got %+v, %v", in, q, err)
		}
	}
	// A plain search mentioning tags is still a search.
	if q, err := ParseQuery("tagging tips"); err != nil || q.Tag != "" {
		t.Fatalf("got %+v, %v", q, err)
	}
}

func TestCacheReusesFreshResultsAndKeepsPostsOnError(t *testing.T) {
	c := NewCache(time.Minute)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	calls := 0
	good := func() ([]*connector.RemotePost, error) {
		calls++
		return []*connector.RemotePost{{RemoteID: "1"}}, nil
	}

	c.Get(1, 1, "k", now, good)
	c.Get(1, 1, "k", now.Add(30*time.Second), good)
	if calls != 1 {
		t.Fatalf("fresh result should be reused, fetched %d times", calls)
	}

	c.Get(1, 1, "other", now.Add(31*time.Second), good)
	if calls != 2 {
		t.Fatal("a changed key must refetch")
	}

	r := c.Get(1, 1, "other", now.Add(2*time.Minute), func() ([]*connector.RemotePost, error) {
		return nil, errors.New("boom")
	})
	if r.Err == nil || len(r.Posts) != 1 {
		t.Fatalf("expected error with last good posts, got %+v", r)
	}
}

func TestCacheHonoursRateLimit(t *testing.T) {
	c := NewCache(time.Minute)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	calls := 0
	limited := func() ([]*connector.RemotePost, error) {
		calls++
		return nil, &connector.RateLimitedError{RetryAfter: 10 * time.Minute}
	}

	r := c.Get(1, 1, "k", now, limited)
	if !r.RetryAfter.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("unexpected retry after %v", r.RetryAfter)
	}
	c.Get(1, 1, "k", now.Add(5*time.Minute), limited)
	if calls != 1 {
		t.Fatal("must not refetch while rate limited")
	}
	c.Get(1, 1, "k", now.Add(11*time.Minute), limited)
	if calls != 2 {
		t.Fatal("should refetch once the rate limit expires")
	}
}

func TestCacheSeparatesAccountsAndForgetClearsAll(t *testing.T) {
	c := NewCache(time.Minute)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	calls := 0
	fetch := func() ([]*connector.RemotePost, error) {
		calls++
		return nil, nil
	}

	c.Get(1, 10, "k", now, fetch)
	c.Get(1, 20, "k", now, fetch)
	if calls != 2 {
		t.Fatalf("each account should fetch, got %d", calls)
	}

	c.Forget(1)
	c.Get(1, 10, "k", now, fetch)
	c.Get(1, 20, "k", now, fetch)
	if calls != 4 {
		t.Fatalf("Forget should clear every account, got %d fetches", calls)
	}
}

func TestMerge(t *testing.T) {
	at := func(m int) time.Time { return time.Date(2026, 10, 1, 12, m, 0, 0, time.UTC) }
	merged := Merge(map[uint32][]*connector.RemotePost{
		1: {{RemoteURL: "u/a", PostedAt: at(1)}, {RemoteURL: "u/shared", PostedAt: at(5)}},
		2: {{RemoteURL: "u/shared", PostedAt: at(5)}, {RemoteURL: "u/b", PostedAt: at(9)}, nil},
	}, []uint32{1, 2}, 0)

	if len(merged) != 3 {
		t.Fatalf("expected 3 posts after de-duplication, got %d", len(merged))
	}
	if merged[0].Post.RemoteURL != "u/b" || merged[1].Post.RemoteURL != "u/shared" || merged[2].Post.RemoteURL != "u/a" {
		t.Fatalf("unexpected order: %v %v %v", merged[0].Post.RemoteURL, merged[1].Post.RemoteURL, merged[2].Post.RemoteURL)
	}
	if merged[1].AccountID != 1 {
		t.Fatal("a shared post should be attributed to the first account")
	}

	if got := Merge(map[uint32][]*connector.RemotePost{1: {{RemoteURL: "x"}, {RemoteURL: "y"}}}, []uint32{1}, 1); len(got) != 1 {
		t.Fatalf("limit not applied: %d", len(got))
	}
}
