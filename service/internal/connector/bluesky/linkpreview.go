package bluesky

import (
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

	client := &http.Client{Timeout: linkPreviewTimeout}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ogUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		return nil, nil
	}

	body := io.LimitReader(resp.Body, linkPreviewMaxBody)
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	preview := &LinkPreview{}
	baseURL := parsed.Scheme + "://" + parsed.Host

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var prop, name, content string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "property":
					prop = a.Val
				case "name":
					name = a.Val
				case "content":
					content = a.Val
				}
			}
			if content == "" {
				return
			}
			prop = strings.ToLower(prop)
			name = strings.ToLower(name)
			switch {
			case prop == "og:title" || name == "twitter:title":
				if preview.Title == "" {
					preview.Title = trimAndTruncate(content, maxTitleLength)
				}
			case prop == "og:description" || name == "twitter:description" || name == "description":
				if preview.Description == "" {
					preview.Description = trimAndTruncate(content, maxDescriptionLength)
				}
			case prop == "og:image" || name == "twitter:image":
				if preview.ImageURL == "" {
					preview.ImageURL = resolveURL(baseURL, content)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Require at least title or description for a valid card; use URL as fallback title
	if preview.Title == "" && preview.Description == "" {
		preview.Title = trimAndTruncate(parsed.Host+parsed.Path, maxTitleLength)
	}
	if preview.Title == "" {
		preview.Title = trimAndTruncate(rawURL, maxTitleLength)
	}

	return preview, nil
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
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}
