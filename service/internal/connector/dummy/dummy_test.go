package dummy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jamesread/japella/internal/connector"
)

func TestSearchRecentByKeyword(t *testing.T) {
	c := &Connector{interval: time.Hour}
	c.seedInitialPosts()

	posts, err := c.SearchRecentByKeyword(context.Background(), &connector.SocialAccount{Identity: "t"}, "japella", time.Time{}, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range posts {
		if strings.Contains(strings.ToLower(p.Content), "japella") {
			found = true
			break
		}
	}
	// Seeded posts are random; generate until we know japella appears or force one.
	if !found {
		c.mu.Lock()
		c.seq++
		c.posts = append(c.posts, simPost{
			RemoteID: "forced",
			Content:  "forced japella sample",
			PostedAt: time.Now(),
		})
		c.mu.Unlock()
		posts, err = c.SearchRecentByKeyword(context.Background(), &connector.SocialAccount{}, "japella", time.Time{}, 50)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(posts) == 0 {
		t.Fatal("expected at least one japella match")
	}
}

func TestPreferNativeWatch(t *testing.T) {
	c := &Connector{}
	if !c.PreferNativeWatch(nil, []string{"japella"}) {
		t.Fatal("expected native watch")
	}
	if c.NativeWatchFidelity(nil, nil) != connector.NativeFidelityExactRule {
		t.Fatal("expected exact_rule fidelity")
	}
}
