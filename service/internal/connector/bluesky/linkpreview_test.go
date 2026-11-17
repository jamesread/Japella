package bluesky

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewServer(t *testing.T, status int, contentType, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestFetchLinkPreview(t *testing.T) {
	b := &BlueskyConnector{}

	cases := []struct {
		name string
		head string
		want LinkPreview
	}{
		{
			name: "open graph tags, relative image",
			head: `<meta property="og:title" content="OG Title"><meta property="og:description" content="OG desc"><meta property="og:image" content="/img/card.png">`,
			want: LinkPreview{Title: "OG Title", Description: "OG desc", ImageURL: "/img/card.png"},
		},
		{
			name: "twitter and description fallbacks",
			head: `<meta name="twitter:title" content="TW Title"><meta name="description" content="Plain desc"><meta name="twitter:image" content="https://cdn.example/x.png">`,
			want: LinkPreview{Title: "TW Title", Description: "Plain desc", ImageURL: "https://cdn.example/x.png"},
		},
		{
			name: "first tag wins, case-insensitive attributes",
			head: `<meta PROPERTY="OG:TITLE" content="First"><meta property="og:title" content="Second"><meta name="description" content=""><meta name="twitter:description" content="Desc">`,
			want: LinkPreview{Title: "First", Description: "Desc"},
		},
		{
			name: "whitespace collapsed",
			head: "<meta property=\"og:title\" content=\"  Spaced \n  out  title \">",
			want: LinkPreview{Title: "Spaced out title"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := previewServer(t, http.StatusOK, "text/html; charset=utf-8", "<html><head>"+c.head+"</head><body></body></html>")
			got, err := b.fetchLinkPreview(url + "/article")
			require.NoError(t, err)
			require.NotNil(t, got)
			want := c.want
			if strings.HasPrefix(want.ImageURL, "/") {
				want.ImageURL = url + want.ImageURL
			}
			assert.Equal(t, want, *got)
		})
	}
}

func TestFetchLinkPreviewFallbackTitle(t *testing.T) {
	url := previewServer(t, http.StatusOK, "text/html", "<html><head><title>ignored</title></head></html>")
	got, err := (&BlueskyConnector{}).fetchLinkPreview(url + "/some/path")
	require.NoError(t, err)
	assert.Equal(t, strings.TrimPrefix(url, "http://")+"/some/path", got.Title)
	assert.Empty(t, got.Description)
}

func TestFetchLinkPreviewTruncates(t *testing.T) {
	long := strings.Repeat("a", maxTitleLength+50)
	url := previewServer(t, http.StatusOK, "text/html", `<meta property="og:title" content="`+long+`">`)
	got, err := (&BlueskyConnector{}).fetchLinkPreview(url)
	require.NoError(t, err)
	assert.Len(t, got.Title, maxTitleLength)
}

func TestFetchLinkPreviewNoPreview(t *testing.T) {
	b := &BlueskyConnector{}
	for name, url := range map[string]string{
		"not found": previewServer(t, http.StatusNotFound, "text/html", `<meta property="og:title" content="x">`),
		"not html":  previewServer(t, http.StatusOK, "application/json", `{"title":"x"}`),
		"ftp":       "ftp://example.com/file",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := b.fetchLinkPreview(url)
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestResolveURL(t *testing.T) {
	base := "https://site.example"
	for _, c := range []struct{ ref, want string }{
		{"", ""},
		{"  /a.png ", "https://site.example/a.png"},
		{"img/b.png", "https://site.example/img/b.png"},
		{"//cdn.example/c.png", "https://cdn.example/c.png"},
		{"https://cdn.example/d", "https://cdn.example/d"},
		{"http://cdn.example/e", "http://cdn.example/e"},
	} {
		assert.Equal(t, c.want, resolveURL(base, c.ref), c.ref)
	}
}
