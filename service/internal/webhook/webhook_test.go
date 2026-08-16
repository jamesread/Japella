package webhook

import (
	"strings"
	"testing"
)

func TestNormalizeEvent(t *testing.T) {
	got, err := NormalizeEvent(" approval.requested ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != EventApprovalRequested {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeEvent("unknown.event"); err == nil {
		t.Fatal("expected error for unknown event")
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"https://example.com/hook", true},
		{"http://127.0.0.1:8080/callback", true},
		{"ftp://example.com/hook", false},
		{"", false},
		{"not-a-url", false},
	}
	for _, tc := range cases {
		_, err := NormalizeURL(tc.in)
		if tc.valid && err != nil {
			t.Fatalf("%q: expected valid, got %v", tc.in, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("%q: expected invalid", tc.in)
		}
	}
}

func TestSignatureHex(t *testing.T) {
	sig := Signature(`{"event":"approval.requested"}`, "secret")
	if len(sig) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(sig))
	}
	if strings.ContainsAny(sig, "ghijklmnopqrstuvwxyzGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("signature is not hex: %q", sig)
	}
}
