package frontend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSPA(t *testing.T) http.Handler {
	t.Helper()
	parent := t.TempDir()
	webui := filepath.Join(parent, "webui")
	require.NoError(t, os.MkdirAll(filepath.Join(webui, "assets"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(webui, "index.html"), []byte("INDEX"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(webui, "assets", "app.js"), []byte("APP"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("SECRET"), 0o600))
	return spaHandler(webui)
}

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	// Set the path directly so it is not cleaned, as a request that bypassed
	// ServeMux's path cleaning would arrive.
	req.URL.Path = path
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	return rr.Code, string(body)
}

func TestSPAHandler(t *testing.T) {
	h := newTestSPA(t)

	cases := []struct {
		path, wantBody string
		wantCode       int
	}{
		{"/", "INDEX", http.StatusOK},
		{"/assets/app.js", "APP", http.StatusOK},
		{"/calendar", "INDEX", http.StatusOK},
		{"/settings/users", "INDEX", http.StatusOK},
		{"/healthz", "404 page not found\n", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			code, body := get(t, h, c.path)
			assert.Equal(t, c.wantCode, code)
			assert.Equal(t, c.wantBody, body)
		})
	}
}

func TestSPAHandlerStaysInsideWebUIDir(t *testing.T) {
	h := newTestSPA(t)
	for _, path := range []string{"/../secret.txt", "/assets/../../secret.txt", "../secret.txt"} {
		_, body := get(t, h, path)
		assert.NotContains(t, body, "SECRET", path)
	}
}
