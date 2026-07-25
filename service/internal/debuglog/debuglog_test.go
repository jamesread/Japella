package debuglog

import (
	"testing"
)

func TestSetAndHelpers(t *testing.T) {
	Init(map[string]bool{
		KeyAuth: false,
		KeyFeed: false,
		KeyHTTP: false,
	})

	if AuthEnabled() || FeedEnabled() || HTTPEnabled() {
		t.Fatal("expected all debug flags off after Init(false)")
	}

	Set(KeyAuth, true)
	if !AuthEnabled() {
		t.Fatal("expected auth enabled")
	}
	if FeedEnabled() || HTTPEnabled() {
		t.Fatal("expected only auth enabled")
	}

	// Helpers must no-op when disabled / not panic when enabled.
	Feedf("should not appear")
	Authf("auth test %s", "ok")
	HTTPf("should not appear")

	Set(KeyHTTP, true)
	if !HTTPEnabled() {
		t.Fatal("expected http enabled")
	}

	if !IsDebugCvar(KeyAuth) || IsDebugCvar("base_url") {
		t.Fatal("IsDebugCvar mismatch")
	}
}
