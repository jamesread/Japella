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
	// SizeBytes is 0 if the file couldn't be stat'd.
	SizeBytes int64
	// OptimisedOf is the original this is an optimised copy of, if it is
	// one, and OptimisedKind which kind (KindLossless or KindPosting).
	OptimisedOf   string
	OptimisedKind string
	// An original's copies, if they exist.
	OptimisedVersion string
	PostingVersion   string
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
	// Originals by stem, and each original stem's copies.
	originals := map[string]string{}
	copies := map[string]map[string]string{}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		if stem, kind := DerivedOf(e.Name()); kind != "" {
			if copies[stem] == nil {
				copies[stem] = map[string]string{}
			}
			copies[stem][kind] = e.Name()
		} else {
			originals[Stem(e.Name())] = e.Name()
		}
	}
	var list []Item
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		item := Item{Filename: e.Name(), URL: "/media/files/" + e.Name()}
		if info, err := e.Info(); err == nil {
			item.SizeBytes = info.Size()
		}
		if stem, kind := DerivedOf(e.Name()); kind != "" {
			item.OptimisedOf, item.OptimisedKind = originals[stem], kind
		} else {
			c := copies[Stem(e.Name())]
			item.OptimisedVersion, item.PostingVersion = c[KindLossless], c[KindPosting]
		}
		list = append(list, item)
	}
	return list, nil
}

// Delete removes a media file from the library by filename.
func Delete(filename string) error {
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		return os.ErrNotExist
	}
	dir, err := GetDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, filepath.Base(filename))
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if absDir != abs && !strings.HasPrefix(abs, absDir+string(os.PathSeparator)) {
		return os.ErrNotExist
	}
	if err := os.Remove(abs); err != nil {
		return err
	}
	return nil
}
