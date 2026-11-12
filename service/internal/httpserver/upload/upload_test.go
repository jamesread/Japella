package upload

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesread/japella/internal/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func multipartUpload(t *testing.T, fields map[string]string, fileField, fileName string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	fw, err := mw.CreateFormFile(fileField, fileName)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadStoresFileInMediaDir(t *testing.T) {
	t.Chdir(t.TempDir())
	dir, err := media.GetDir()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	Handle(rec, multipartUpload(t, map[string]string{"note": "ignored"}, "file", "photo.png", []byte("png-bytes")))

	assert.Equal(t, http.StatusOK, rec.Code)
	matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
	require.NoError(t, err)
	require.Len(t, matches, 1)
	got, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, []byte("png-bytes"), got)
}

func TestUploadWithoutFileFieldIsRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	rec := httptest.NewRecorder()
	Handle(rec, multipartUpload(t, nil, "other", "photo.png", []byte("x")))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUploadRejectsNonMultipart(t *testing.T) {
	t.Chdir(t.TempDir())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", bytes.NewBufferString("plain"))
	Handle(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
