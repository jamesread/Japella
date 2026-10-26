package feedauthor

import "testing"

func TestIsOwnPost(t *testing.T) {
	cases := []struct {
		identity, handle, name string
		own                    bool
	}{
		{"alice@example.social", "@alice@example.social", "Alice", true},
		{"alice", "@alice@example.social", "Alice", true},
		{"@Alice@Example.Social", "@alice", "", true},
		{"alice@example.social", "@alice@other.social", "", false},
		{"alice@example.social", "@bob@example.social", "Alice", false},
		{"alice.bsky.social", "alice.bsky.social", "", true},
		{"alice", "", "alice", true},
		{"alice", "", "bob", false},
		{"alice", "", "", true},
	}
	for _, c := range cases {
		if got := IsOwnPost(c.identity, c.handle, c.name); got != c.own {
			t.Errorf("IsOwnPost(%q, %q, %q) = %v, want %v", c.identity, c.handle, c.name, got, c.own)
		}
	}
}

func TestPostKey(t *testing.T) {
	if PostKey(" https://x/1 ", "9") != "url:https://x/1" {
		t.Fatal("URL should win")
	}
	if PostKey("", "9") != "id:9" || PostKey("", "") != "" {
		t.Fatal("unexpected fallback")
	}
	if KeyHash("url:a") == KeyHash("id:a") || len(KeyHash("x")) != 64 {
		t.Fatal("hashes must differ per key kind and be 64 hex chars")
	}
}
