package chatbot

import "testing"

func TestValidateBotID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"bota", true},
		{"support-bot", true},
		{"a1", true},
		{"", false},
		{"BotA", false},
		{"_", false},
		{"yaml", false},
		{"x", false},
	}
	for _, tc := range tests {
		err := ValidateBotID(tc.id)
		if tc.valid && err != nil {
			t.Errorf("ValidateBotID(%q) unexpected error: %v", tc.id, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("ValidateBotID(%q) expected error", tc.id)
		}
	}
}

func TestControllerKey(t *testing.T) {
	if got := ControllerKey("telegram", "bota"); got != "telegram-bota" {
		t.Fatalf("got %q", got)
	}
}
