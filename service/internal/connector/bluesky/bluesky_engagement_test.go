package bluesky

import "testing"

func TestParseBlueskyPostRef(t *testing.T) {
	cases := []struct {
		in, repo, rkey string
		ok             bool
	}{
		{"https://bsky.app/profile/alice.bsky.social/post/3kabc123", "alice.bsky.social", "3kabc123", true},
		{"https://bsky.app/profile/did:plc:xyz/post/3kabc123", "did:plc:xyz", "3kabc123", true},
		{"at://did:plc:xyz/app.bsky.feed.post/3kabc123", "did:plc:xyz", "3kabc123", true},
		{"at://did:plc:xyz/app.bsky.feed.like/3kabc123", "", "", false},
		{"https://bsky.app/profile/alice.bsky.social", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		repo, rkey, err := parseBlueskyPostRef(c.in)
		if c.ok != (err == nil) {
			t.Fatalf("%q: err = %v, want ok=%v", c.in, err, c.ok)
		}
		if repo != c.repo || rkey != c.rkey {
			t.Fatalf("%q: got (%q, %q), want (%q, %q)", c.in, repo, rkey, c.repo, c.rkey)
		}
	}
}
