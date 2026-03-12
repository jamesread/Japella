package media

import (
	"os"
	"path/filepath"
	"strings"
)

const mediaFilesPrefix = "/media/files/"

// GetDir returns the media storage directory, creating it if needed.
// It tries several candidate paths and returns the first that is writable.
func GetDir() (string, error) {
	dirs := []string{"./media", "../var/media", "/app/media"}
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(abs, 0755); err != nil {
			continue
		}
		f, err := os.CreateTemp(abs, ".write-test-")
		if err != nil {
			continue
		}
		f.Close()
		os.Remove(f.Name())
		return abs, nil
	}
	abs, _ := filepath.Abs("media")
	return abs, os.MkdirAll(abs, 0755)
}

// Item describes a single media file for listing.
type Item struct {
	Filename string
	URL      string
}

// PathFromURL resolves a media URL (e.g. /media/files/filename.png) to an absolute path
// in the media directory. Returns an error if the URL is not a valid /media/files/ URL
// or the file does not exist.
func PathFromURL(mediaURL string) (string, error) {
	if !strings.HasPrefix(mediaURL, mediaFilesPrefix) {
		return "", os.ErrNotExist
	}
	filename := strings.TrimPrefix(mediaURL, mediaFilesPrefix)
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		return "", os.ErrNotExist
	}
	dir, err := GetDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filepath.Base(filename))
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if absDir != abs && !strings.HasPrefix(abs, absDir+string(os.PathSeparator)) {
		return "", os.ErrNotExist
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// List returns all media files in the media directory.
// URL is the path to fetch the file (e.g. /media/files/<filename>).
func List() ([]Item, error) {
	dir, err := GetDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var list []Item
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		list = append(list, Item{Filename: e.Name(), URL: "/media/files/" + e.Name()})
	}
	return list, nil
}
