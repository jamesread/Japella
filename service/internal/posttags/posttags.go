// Package posttags normalises the internal tags users put on posts. Tags
// are private to Japella; they are never posted anywhere.
package posttags

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// MaxLength matches post_tags.tag.
	MaxLength = 64
	// MaxPerPost limits how many tags one post can carry.
	MaxPerPost = 20
)

var tagRe = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

// Normalise lowercases a tag and drops a leading '#', so "#ToDo" and "todo"
// are the same tag.
func Normalise(raw string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(raw))
	t = strings.TrimLeft(t, "#")
	if t == "" {
		return "", fmt.Errorf("tag is empty")
	}
	if utf8.RuneCountInString(t) > MaxLength {
		return "", fmt.Errorf("tag %q is longer than %d characters", t, MaxLength)
	}
	if !tagRe.MatchString(t) {
		return "", fmt.Errorf("tag %q may only contain letters, numbers, '-' and '_'", t)
	}
	return t, nil
}

// NormaliseList normalises tags, dropping duplicates and keeping their order.
func NormaliseList(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, r := range raw {
		t, err := Normalise(r)
		if err != nil {
			return nil, err
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) > MaxPerPost {
		return nil, fmt.Errorf("a post can have at most %d tags", MaxPerPost)
	}
	return out, nil
}
