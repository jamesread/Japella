package posttags

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalise(t *testing.T) {
	cases := map[string]string{
		"todo":        "todo",
		"  #ToDo ":    "todo",
		"##follow-up": "follow-up",
		"read_later":  "read_later",
		"été":         "été",
	}
	for in, want := range cases {
		got, err := Normalise(in)
		if err != nil || got != want {
			t.Errorf("Normalise(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	for _, bad := range []string{"", "#", "two words", "a/b", "<b>", strings.Repeat("x", MaxLength+1)} {
		if _, err := Normalise(bad); err == nil {
			t.Errorf("Normalise(%q) should fail", bad)
		}
	}
}

func TestNormaliseList(t *testing.T) {
	got, err := NormaliseList([]string{"Todo", "#todo", "later"})
	if err != nil || !reflect.DeepEqual(got, []string{"todo", "later"}) {
		t.Fatalf("NormaliseList = %v, %v", got, err)
	}

	if _, err := NormaliseList([]string{"ok", "not ok"}); err == nil {
		t.Error("NormaliseList should reject an invalid tag")
	}

	many := make([]string, MaxPerPost+1)
	for i := range many {
		many[i] = strings.Repeat("a", i+1)
	}
	if _, err := NormaliseList(many); err == nil {
		t.Error("NormaliseList should reject too many tags")
	}
}
