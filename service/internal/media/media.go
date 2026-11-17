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
		if err = os.MkdirAll(abs, 0o750); err != nil {
			continue
		}
		f, err := os.CreateTemp(abs, ".write-test-")
		if err != nil {
			continue
		}
		_ = f.Close()
		_ = os.Remove(f.Name())
		return abs, nil
	}
	abs, _ := filepath.Abs("media")
	return abs, os.MkdirAll(abs, 0o750)
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

func isUnsafeFilename(filename string) bool {
	return filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..")
}

func confinedToDir(abs, absDir string) bool {
	return absDir == abs || strings.HasPrefix(abs, absDir+string(os.PathSeparator))
}

func resolveInMediaDir(filename string) (string, error) {
	dir, err := GetDir()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Join(dir, filepath.Base(filename)))
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if !confinedToDir(abs, absDir) {
		return "", os.ErrNotExist
	}
	return abs, nil
}

// PathFromURL resolves a media URL (e.g. /media/files/filename.png) to an absolute path
// in the media directory. Returns an error if the URL is not a valid /media/files/ URL
// or the file does not exist.
func PathFromURL(mediaURL string) (string, error) {
	if !strings.HasPrefix(mediaURL, mediaFilesPrefix) {
		return "", os.ErrNotExist
	}
	filename := strings.TrimPrefix(mediaURL, mediaFilesPrefix)
	if isUnsafeFilename(filename) {
		return "", os.ErrNotExist
	}
	abs, err := resolveInMediaDir(filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func skipIndexEntry(e os.DirEntry) bool {
	return e.IsDir() || strings.HasSuffix(e.Name(), ".tmp")
}

func indexOneEntry(e os.DirEntry, originals map[string]string, copies map[string]map[string]string) {
	if stem, kind := DerivedOf(e.Name()); kind != "" {
		if copies[stem] == nil {
			copies[stem] = map[string]string{}
		}
		copies[stem][kind] = e.Name()
		return
	}
	originals[Stem(e.Name())] = e.Name()
}

func indexMediaEntries(entries []os.DirEntry) (originals map[string]string, copies map[string]map[string]string) {
	originals = map[string]string{}
	copies = map[string]map[string]string{}
	for _, e := range entries {
		if skipIndexEntry(e) {
			continue
		}
		indexOneEntry(e, originals, copies)
	}
	return originals, copies
}

func listItemFromEntry(e os.DirEntry, originals map[string]string, copies map[string]map[string]string) Item {
	item := Item{Filename: e.Name(), URL: "/media/files/" + e.Name()}
	if info, err := e.Info(); err == nil {
		item.SizeBytes = info.Size()
	}
	if stem, kind := DerivedOf(e.Name()); kind != "" {
		item.OptimisedOf, item.OptimisedKind = originals[stem], kind
		return item
	}
	c := copies[Stem(e.Name())]
	item.OptimisedVersion, item.PostingVersion = c[KindLossless], c[KindPosting]
	return item
}

func skipListEntry(e os.DirEntry) bool {
	return e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasSuffix(e.Name(), ".tmp")
}

func buildMediaList(entries []os.DirEntry) []Item {
	originals, copies := indexMediaEntries(entries)
	var list []Item
	for _, e := range entries {
		if skipListEntry(e) {
			continue
		}
		list = append(list, listItemFromEntry(e, originals, copies))
	}
	return list
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
	return buildMediaList(entries), nil
}

// Delete removes a media file from the library by filename.
func Delete(filename string) error {
	if isUnsafeFilename(filename) {
		return os.ErrNotExist
	}
	abs, err := resolveInMediaDir(filename)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}
