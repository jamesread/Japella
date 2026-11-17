package upload

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// media.GetDir looks for ./media first, so tests run in a temporary directory.
func inTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	return filepath.Join(dir, "media")
}

func uploadRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func mediaFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestUploadStoresFile(t *testing.T) {
	dir := inTempDir(t)
	rr := httptest.NewRecorder()
	Handle(rr, uploadRequest(t, "photo.png", []byte("png-data")))

	assert.Equal(t, http.StatusOK, rr.Code)
	files := mediaFiles(t, dir)
	require.Len(t, files, 1)
	assert.Equal(t, ".png", filepath.Ext(files[0]))
	data, err := os.ReadFile(filepath.Join(dir, files[0]))
	require.NoError(t, err)
	assert.Equal(t, []byte("png-data"), data)
}

func TestUploadRejectsOversizedBody(t *testing.T) {
	dir := inTempDir(t)
	rr := httptest.NewRecorder()
	Handle(rr, uploadRequest(t, "big.png", bytes.Repeat([]byte("x"), maxUploadSize+1)))

	assert.GreaterOrEqual(t, rr.Code, 400)
	assert.Empty(t, mediaFiles(t, dir))
}

func TestServeMedia(t *testing.T) {
	dir := inTempDir(t)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.png"), []byte("A"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(dir), "secret.txt"), []byte("SECRET"), 0o600))

	serve := func(path string) (int, string) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.URL.Path = path
		rr := httptest.NewRecorder()
		HandleServeMedia(rr, req)
		body, err := io.ReadAll(rr.Body)
		require.NoError(t, err)
		return rr.Code, string(body)
	}

	code, body := serve("/media/files/a.png")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "A", body)

	for _, path := range []string{"/media/files/../secret.txt", "/media/files/..%2fsecret.txt", "/media/files/missing.png"} {
		code, body = serve(path)
		assert.Equal(t, http.StatusNotFound, code, path)
		assert.NotContains(t, body, "SECRET", path)
	}
}
