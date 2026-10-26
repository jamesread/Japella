package utils

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlLineBreakRe = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlParagraphRe = regexp.MustCompile(`(?i)</p>\s*<p[^>]*>`)
	htmlTagRe       = regexp.MustCompile(`<[^>]*>`)
	blankRunRe      = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText turns post HTML (as Mastodon sends it) into plain text, keeping
// line and paragraph breaks.
func HTMLToText(s string) string {
	s = htmlLineBreakRe.ReplaceAllString(s, "\n")
	s = htmlParagraphRe.ReplaceAllString(s, "\n\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(blankRunRe.ReplaceAllString(s, "\n\n"))
}

// TextToSafeHTML escapes plain text for v-html rendering, keeping line breaks.
func TextToSafeHTML(s string) string {
	return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
}
