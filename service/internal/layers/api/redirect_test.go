package api

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectEscapesMessageInPage(t *testing.T) {
	rec := httptest.NewRecorder()

	redirect("https://japella.example", rec, `OAuth2 error: <script>alert(1)</script>"`, "bad")

	body := rec.Body.String()
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.Contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func TestRedirectEncodesNotificationInLocation(t *testing.T) {
	rec := httptest.NewRecorder()

	redirect("https://japella.example", rec, `bad" onload="x & y=z`, "bad")

	location := rec.Header().Get("Location")
	require.True(t, strings.HasPrefix(location, "https://japella.example/?"), location)

	parsed, err := url.Parse(location)
	require.NoError(t, err)
	assert.Equal(t, `bad" onload="x & y=z`, parsed.Query().Get("notification"))
	assert.Equal(t, "bad", parsed.Query().Get("type"))
	assert.NotContains(t, rec.Body.String(), `onload="x`)
}
