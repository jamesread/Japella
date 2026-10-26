// Package languages normalises post language codes and decides whether a post
// passes a social account's preferred languages.
package languages

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// MaxPreferred caps how many languages an account can prefer.
const MaxPreferred = 20

var codeRe = regexp.MustCompile(`^[a-z]{2,3}$`)

// Base reduces a language tag ("en", "EN", "pt-BR", "zh_Hant") to its
// lowercase base language. It returns "" for missing or non-language codes,
// such as X's "und" (undetermined), "zxx" (no linguistic content) and its
// private q* codes.
func Base(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		tag = tag[:i]
	}
	if !codeRe.MatchString(tag) || tag == "und" || tag == "zxx" || tag == "mul" || tag[0] == 'q' {
		return ""
	}
	return tag
}

// ParsePreferred validates a list of language codes, returning them
// normalised, de-duplicated and sorted.
func ParsePreferred(codes []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(codes))
	for _, raw := range codes {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		code := Base(raw)
		if code == "" {
			return nil, fmt.Errorf("%q is not a language code; use ISO 639 codes like en, de or pt", strings.TrimSpace(raw))
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	if len(out) > MaxPreferred {
		return nil, fmt.Errorf("at most %d preferred languages are allowed", MaxPreferred)
	}
	sort.Strings(out)
	return out, nil
}

// Join stores a parsed list as a comma-separated string.
func Join(codes []string) string {
	return strings.Join(codes, ",")
}

// Split reads a stored comma-separated list.
func Split(stored string) []string {
	out := make([]string, 0)
	for _, c := range strings.Split(stored, ",") {
		if c = Base(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// Filter is a set of preferred languages. The zero value allows everything.
type Filter map[string]bool

// NewFilter builds a filter from a stored comma-separated list.
func NewFilter(stored string) Filter {
	f := Filter{}
	for _, c := range Split(stored) {
		f[c] = true
	}
	return f
}

// Allows reports whether a post in language tag passes. With no preferences
// set everything passes, and posts whose language is unknown always pass:
// there is nothing to judge them on.
func (f Filter) Allows(tag string) bool {
	if len(f) == 0 {
		return true
	}
	base := Base(tag)
	return base == "" || f[base]
}
