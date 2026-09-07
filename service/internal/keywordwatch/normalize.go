package keywordwatch

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var spaceRe = regexp.MustCompile(`\s+`)

const (
	MaxKeywordsPerAccount = 20
	MinKeywordLength      = 2
	MaxKeywordLength      = 128
	MaxContentSnippet     = 2000
)

// NormalizeKeyword returns (display, normalized, error).
func NormalizeKeyword(raw string) (string, string, error) {
	display := strings.TrimSpace(raw)
	display = spaceRe.ReplaceAllString(display, " ")
	if display == "" {
		return "", "", errKeyword("keyword must not be empty")
	}
	if len([]rune(display)) < MinKeywordLength {
		return "", "", errKeyword("keyword is too short")
	}
	if len([]rune(display)) > MaxKeywordLength {
		return "", "", errKeyword("keyword is too long")
	}
	norm := strings.ToLower(display)
	if isDeniedKeyword(norm) {
		return "", "", errKeyword("keyword is too common to watch")
	}
	return display, norm, nil
}

func isDeniedKeyword(norm string) bool {
	switch norm {
	case "a", "an", "the", "and", "or", "to", "of", "in", "on", "is", "it", "i", "you", "for":
		return true
	}
	return false
}

type keywordError string

func (e keywordError) Error() string { return string(e) }

func errKeyword(msg string) error { return keywordError(msg) }

// StripHTML removes tags and unescapes entities for match verification.
func StripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

// ContainsKeyword reports whether content contains keyword (case-insensitive substring).
func ContainsKeyword(content, keyword string) bool {
	c := strings.ToLower(StripHTML(content))
	k := strings.ToLower(strings.TrimSpace(keyword))
	if k == "" {
		return false
	}
	return strings.Contains(c, k)
}

// IsHashtagKeyword reports whether the keyword is a hashtag (optional leading #).
func IsHashtagKeyword(keyword string) bool {
	k := strings.TrimSpace(keyword)
	if k == "" {
		return false
	}
	if strings.HasPrefix(k, "#") {
		k = k[1:]
	}
	if k == "" {
		return false
	}
	for _, r := range k {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

// HashtagTag returns the tag without leading #.
func HashtagTag(keyword string) string {
	k := strings.TrimSpace(keyword)
	return strings.TrimPrefix(k, "#")
}

// TruncateSnippet caps stored/dispatched content length.
func TruncateSnippet(s string) string {
	s = StripHTML(s)
	runes := []rune(s)
	if len(runes) <= MaxContentSnippet {
		return s
	}
	return string(runes[:MaxContentSnippet])
}
