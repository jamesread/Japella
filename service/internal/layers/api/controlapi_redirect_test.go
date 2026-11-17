package api

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OAuth2 callback query parameters reach redirect() as the message, so it must
// treat the message as untrusted.
func TestRedirectEscapesMessage(t *testing.T) {
	t.Setenv("JAPELLA_DEV_REDIRECT_VITE", "")
	message := `OAuth2 error: "><script>alert(1)</script>&type=good`

	rr := httptest.NewRecorder()
	redirect("https://japella.example", rr, message, "bad")

	body := rr.Body.String()
	assert.NotContains(t, body, "<script>")
	assert.NotContains(t, body, `"><`)
	assert.Contains(t, body, "&lt;script&gt;")

	location, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "japella.example", location.Host)
	assert.Equal(t, message, location.Query().Get("notification"))
	assert.Equal(t, "bad", location.Query().Get("type"))
	assert.True(t, strings.HasPrefix(rr.Header().Get("Location"), "https://japella.example/?"))
}
