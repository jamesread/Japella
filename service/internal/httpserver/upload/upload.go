package upload

import (
	"io"
	"mime/multipart"
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
	file, header, ok := parseUploadFile(w, r)
	if !ok {
		return
	}
	defer func() { _ = file.Close() }()

	root, ext, ok := openMediaRootForUpload(w, header.Filename)
	if !ok {
		return
	}
	defer func() { _ = root.Close() }()

	if err := saveUploadedFile(root, file, ext); err != nil {
		log.Errorf("write media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("File uploaded successfully"))
}

func parseUploadFile(w http.ResponseWriter, r *http.Request) (multipart.File, *multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+multipartOverhead)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil { //nolint:gosec // body is bounded by http.MaxBytesReader above
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return nil, nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return nil, nil, false
	}
	if header.Size > maxUploadSize {
		_ = file.Close()
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return nil, nil, false
	}
	return file, header, true
}

func openMediaRootForUpload(w http.ResponseWriter, filename string) (*os.Root, string, bool) {
	dir, err := media.GetDir()
	if err != nil {
		log.Errorf("media dir: %v", err)
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return nil, "", false
	}
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Errorf("open media dir: %v", err)
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return nil, "", false
	}
	return root, ext, true
}

func saveUploadedFile(root *os.Root, file multipart.File, ext string) error {
	name := uuid.New().String() + ext
	dst, err := root.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, file)
	if closeErr := dst.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(name)
		return err
	}
	return nil
}

// HandleServeMedia serves a single file under GET /media/files/<filename>.
func HandleServeMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filename, ok := mediaFilenameFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	f, info, ok := openMediaFile(w, r, filename)
	if !ok {
		return
	}
	defer func() { _ = f.Close() }()
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func mediaFilenameFromPath(urlPath string) (string, bool) {
	filename := strings.TrimPrefix(urlPath, "/media/files/")
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		return "", false
	}
	return filename, true
}

func openMediaRoot(w http.ResponseWriter) (*os.Root, bool) {
	dir, err := media.GetDir()
	if err != nil {
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return nil, false
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		http.Error(w, "Failed to access media directory", http.StatusInternalServerError)
		return nil, false
	}
	return root, true
}

func mediaOpenError(w http.ResponseWriter, r *http.Request, err error) (http.File, os.FileInfo, bool) {
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return nil, nil, false
	}
	http.Error(w, "Failed to open file", http.StatusInternalServerError)
	return nil, nil, false
}

func openMediaFile(w http.ResponseWriter, r *http.Request, filename string) (http.File, os.FileInfo, bool) {
	root, ok := openMediaRoot(w)
	if !ok {
		return nil, nil, false
	}
	f, err := root.Open(filename)
	_ = root.Close()
	if err != nil {
		return mediaOpenError(w, r, err)
	}
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		_ = f.Close()
		http.NotFound(w, r)
		return nil, nil, false
	}
	return f, info, true
}
