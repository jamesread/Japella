package upload

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jamesread/japella/internal/media"
	log "github.com/sirupsen/logrus"
)

const maxUploadSize = 10 << 20 // 10MB

// Room for multipart boundaries and headers around a file of maxUploadSize.
const multipartOverhead = 1 << 20

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handleFileUpload(w, r)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+multipartOverhead)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil { //nolint:gosec // body is bounded by http.MaxBytesReader above
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	if header.Size > maxUploadSize {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	dir, err := media.GetDir()
	if err != nil {
		log.Errorf("media dir: %v", err)
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return
	}
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Errorf("open media dir: %v", err)
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return
	}
	defer func() { _ = root.Close() }()

	name := uuid.New().String() + ext
	dst, err := root.Create(name)
	if err != nil {
		log.Errorf("create media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(dst, file)
	if closeErr := dst.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(name)
		log.Errorf("write media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("File uploaded successfully"))
}

// HandleServeMedia serves a single file under GET /media/files/<filename>.
func HandleServeMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filename := strings.TrimPrefix(r.URL.Path, "/media/files/")
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	dir, err := media.GetDir()
	if err != nil {
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
