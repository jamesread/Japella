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

// The largest video any connector accepts is 512 MiB; the extra MiB leaves
// room for multipart headers.
const maxUploadBodyBytes = 513 << 20

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handleFileUpload(w, r)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)
	file, err := uploadedFilePart(r)
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

	ext := filepath.Ext(file.FileName())
	if ext == "" || !media.ValidFilename(ext) {
		ext = ".bin"
	}
	name := uuid.New().String() + ext
	dst, err := media.CreateFile(name)
	if err != nil {
		log.Errorf("create media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	if err := copyAndClose(dst, file); err != nil {
		_ = media.RemoveFile(name)
		log.Errorf("write media file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("File uploaded successfully")); err != nil {
		log.Warnf("write upload response: %v", err)
	}
}

// HandleServeMedia serves a single file under GET /media/files/<filename>.
func HandleServeMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filename := strings.TrimPrefix(r.URL.Path, "/media/files/")
	if !media.ValidFilename(filename) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	f, err := media.OpenFile(filename)
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

// A failed Close on the destination can mean the data never reached disk, so
// it is reported like a failed write.
func copyAndClose(dst *os.File, src io.Reader) error {
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// The file part is streamed straight to disk rather than parsed into a form,
// so nothing is buffered in memory or a temporary file first.
func uploadedFilePart(r *http.Request) (*multipart.Part, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}
	for {
		part, err := reader.NextPart()
		if err != nil {
			return nil, err
		}
		if part.FormName() == "file" && part.FileName() != "" {
			return part, nil
		}
		_ = part.Close()
	}
}
