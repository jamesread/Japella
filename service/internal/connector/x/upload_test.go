package x

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

type uploadRequest struct {
	auth, filename, content, category string
}

// fakeUpload answers media uploads with status and body, and records the request.
func fakeUpload(t *testing.T, status int, body string) *uploadRequest {
	t.Helper()
	got := &uploadRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/2/media/upload", r.URL.Path)
		got.auth = r.Header.Get("Authorization")
		if f, h, err := r.FormFile("media"); err == nil {
			data, _ := io.ReadAll(f)
			got.filename, got.content = h.Filename, string(data)
		}
		got.category = r.FormValue("media_category")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	orig := xAPIBase
	xAPIBase = srv.URL
	t.Cleanup(func() { xAPIBase = orig })
	return got
}

func mediaFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte("bytes of "+name), 0o600))
	return path
}

func TestUploadMediaSendsFile(t *testing.T) {
	got := fakeUpload(t, 200, `{"data":{"media_key":"key1"}}`)
	id, err := (&XConnector{}).uploadMedia(mediaFile(t, "photo.JPG"), "tok")
	require.NoError(t, err)
	assert.Equal(t, "key1", id)
	assert.Equal(t, &uploadRequest{auth: "Bearer tok", filename: "photo.JPG", content: "bytes of photo.JPG", category: "tweet_image"}, got)

	got = fakeUpload(t, 200, `{"data":{"media_key":"key2"}}`)
	_, err = (&XConnector{}).uploadMedia(mediaFile(t, "clip.mp4"), "tok")
	require.NoError(t, err)
	assert.Empty(t, got.category, "only images are marked tweet_image")
}

func TestUploadMediaResponseIDs(t *testing.T) {
	cases := map[string]string{
		`{"data":{"media_key":"k","media_id":"9"}}`: "k",
		`{"data":{"media_id":"7"}}`:                 "7",
		`{"data":{"media_id":123}}`:                 "123",
		`{"media_id_string":"s1","media_id":5}`:     "s1",
		`{"media_id":456}`:                          "456",
	}
	for body, want := range cases {
		fakeUpload(t, 200, body)
		id, err := (&XConnector{}).uploadMedia(mediaFile(t, "a.png"), "tok")
		require.NoError(t, err, body)
		assert.Equal(t, want, id, body)
	}
}

func TestUploadMediaErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{403, "nope", "media upload forbidden (403): X requires the media.write scope. Disconnect and reconnect your X account in Japella so the new permission is granted. Response: nope"},
		{500, "boom", "media upload returned 500: boom"},
		{200, "not json", "decode response: "},
		{200, `{"data":{}}`, "media upload response missing media_id / media_key"},
	}
	for _, c := range cases {
		fakeUpload(t, c.status, c.body)
		_, err := (&XConnector{}).uploadMedia(mediaFile(t, "a.png"), "tok")
		require.Error(t, err)
		assert.Contains(t, err.Error(), c.want)
	}

	_, err := (&XConnector{}).uploadMedia(filepath.Join(t.TempDir(), "missing.png"), "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open file: ")
}

// X media IDs are 64-bit; they must not pass through float64.
func TestUploadMediaKeepsLargeNumericIDs(t *testing.T) {
	for body, want := range map[string]string{
		`{"data":{"media_id":1880028106020515841}}`: "1880028106020515841",
		`{"media_id":1880028106020515843}`:          "1880028106020515843",
	} {
		fakeUpload(t, 200, body)
		id, err := (&XConnector{}).uploadMedia(mediaFile(t, "a.png"), "tok")
		require.NoError(t, err)
		assert.Equal(t, want, id)
	}
}
