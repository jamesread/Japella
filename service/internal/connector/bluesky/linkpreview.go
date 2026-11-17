package bluesky

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// LinkPreview holds Open Graph / meta data for a URL (for rich link cards).
type LinkPreview struct {
	Title       string
	Description string
	ImageURL    string
}

const (
	// Bluesky external embed limits (conservative)
	maxTitleLength       = 256
	maxDescriptionLength = 1000
	linkPreviewTimeout   = 10 * time.Second
	linkPreviewMaxBody   = 512 * 1024 // 512KB
	ogUserAgent          = "Japella/1.0 (link preview; https://github.com/jamesread/japella)"
)

// fetchLinkPreview fetches the given URL and extracts Open Graph (and fallback) meta tags.
// Used to build app.bsky.embed.external rich link cards on Bluesky.
func (b *BlueskyConnector) fetchLinkPreview(rawURL string) (*LinkPreview, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, err
	}

	doc, err := fetchHTML(rawURL)
	if doc == nil {
		return nil, err
	}

	preview := extractLinkPreview(doc, parsed.Scheme+"://"+parsed.Host)
	preview.fillFallbackTitle(parsed, rawURL)
	return preview, nil
}

// fetchHTML returns nil without an error when the page is not a successful
// HTML response, so there is simply no preview.
func fetchHTML(rawURL string) (*html.Node, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ogUserAgent)

	resp, err := (&http.Client{Timeout: linkPreviewTimeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if !isHTMLResponse(resp) {
		return nil, nil
	}
	return html.Parse(io.LimitReader(resp.Body, linkPreviewMaxBody))
}

func isHTMLResponse(resp *http.Response) bool {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}

// Preview fields, in the order a tag naming two of them resolves to.
const (
	previewFieldNone = iota
	previewFieldTitle
	previewFieldDescription
	previewFieldImage
)

var (
	metaPropertyFields = map[string]int{
		"og:title":       previewFieldTitle,
		"og:description": previewFieldDescription,
		"og:image":       previewFieldImage,
	}
	metaNameFields = map[string]int{
		"twitter:title":       previewFieldTitle,
		"twitter:description": previewFieldDescription,
		"description":         previewFieldDescription,
		"twitter:image":       previewFieldImage,
	}
)

func extractLinkPreview(doc *html.Node, baseURL string) *LinkPreview {
	preview := &LinkPreview{}
	for n := range doc.Descendants() {
		if n.Type == html.ElementNode && n.Data == "meta" {
			prop, name, content := metaAttrs(n)
			preview.setOnce(metaField(prop, name), content, baseURL)
		}
	}
	return preview
}

func metaAttrs(n *html.Node) (prop, name, content string) {
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "property":
			prop = strings.ToLower(a.Val)
		case "name":
			name = strings.ToLower(a.Val)
		case "content":
			content = a.Val
		}
	}
	return prop, name, content
}

func metaField(prop, name string) int {
	byProp, byName := metaPropertyFields[prop], metaNameFields[name]
	if byProp == previewFieldNone || (byName != previewFieldNone && byName < byProp) {
		return byName
	}
	return byProp
}

// setOnce keeps the first value found for each field.
func (p *LinkPreview) setOnce(field int, content, baseURL string) {
	if content == "" {
		return
	}
	switch field {
	case previewFieldTitle:
		setIfEmpty(&p.Title, trimAndTruncate(content, maxTitleLength))
	case previewFieldDescription:
		setIfEmpty(&p.Description, trimAndTruncate(content, maxDescriptionLength))
	case previewFieldImage:
		setIfEmpty(&p.ImageURL, resolveURL(baseURL, content))
	}
}

func setIfEmpty(dst *string, value string) {
	if *dst == "" {
		*dst = value
	}
}

// A card needs a title, so fall back to the URL.
func (p *LinkPreview) fillFallbackTitle(parsed *url.URL, rawURL string) {
	if p.Title == "" && p.Description == "" {
		p.Title = trimAndTruncate(parsed.Host+parsed.Path, maxTitleLength)
	}
	if p.Title == "" {
		p.Title = trimAndTruncate(rawURL, maxTitleLength)
	}
}

func trimAndTruncate(s string, max int) string {
	s = strings.TrimSpace(s)
	// collapse newlines
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)
	if max > 0 && len(s) > max {
		s = s[:max]
	}
	return s
}

func resolveURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	refURL, err := url.Parse(ref)
	if ref == "" || err != nil || refURL.IsAbs() {
		return ref
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}
