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

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handleFileUpload(w, r)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

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
	name := uuid.New().String() + ext
	path := filepath.Join(dir, name)
	dst, err := os.Create(path)
	if err != nil {
		log.Errorf("create media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(path)
		log.Errorf("write media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
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
	path := filepath.Join(dir, filepath.Base(filename))
	absPath, err := filepath.Abs(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if absDir != absPath && !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	path = absPath
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
