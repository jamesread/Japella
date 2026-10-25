package x

import "testing"

func TestExtractXTweetID(t *testing.T) {
	cases := []struct {
		remoteID, postURL, want string
		ok                      bool
	}{
		{"1234567890", "", "1234567890", true},
		{"", "https://x.com/user/status/1234567890", "1234567890", true},
		{"not-a-number", "https://x.com/user/status/42", "42", true},
		{"", "https://x.com/user", "", false},
	}
	for _, c := range cases {
		got, err := extractXTweetID(c.remoteID, c.postURL)
		if c.ok != (err == nil) {
			t.Fatalf("(%q, %q): err = %v, want ok=%v", c.remoteID, c.postURL, err, c.ok)
		}
		if got != c.want {
			t.Fatalf("(%q, %q): got %q, want %q", c.remoteID, c.postURL, got, c.want)
		}
	}
}
