package media

import (
	"os"
	"strings"
)

// Media files are accessed through os.Root so a name can never resolve to a
// file outside the media directory, including through a symlink.

func ValidFilename(name string) bool {
	return name != "" && !strings.Contains(name, "/") && !strings.Contains(name, "..")
}

func withRoot[T any](fn func(*os.Root) (T, error)) (T, error) {
	var zero T
	dir, err := GetDir()
	if err != nil {
		return zero, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return zero, err
	}
	defer func() { _ = root.Close() }()
	return fn(root)
}

func OpenFile(name string) (*os.File, error) {
	return withRoot(func(root *os.Root) (*os.File, error) {
		return root.Open(name)
	})
}

func readFile(name string) ([]byte, error) {
	return withRoot(func(root *os.Root) ([]byte, error) {
		return root.ReadFile(name)
	})
}

func CreateFile(name string) (*os.File, error) {
	return withRoot(func(root *os.Root) (*os.File, error) {
		return root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	})
}

func RemoveFile(name string) error {
	_, err := withRoot(func(root *os.Root) (struct{}, error) {
		return struct{}{}, root.Remove(name)
	})
	return err
}

func writeFileAtomic(name string, data []byte) error {
	_, err := withRoot(func(root *os.Root) (struct{}, error) {
		tmp := name + ".tmp"
		if err := root.WriteFile(tmp, data, 0o600); err != nil {
			return struct{}{}, err
		}
		if err := root.Rename(tmp, name); err != nil {
			_ = root.Remove(tmp)
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}
