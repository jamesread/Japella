package media

import (
	"image"
	"os"

	// Decoders for image.DecodeConfig.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Info is a media file's size and, for images Japella can read, dimensions.
type Info struct {
	Filename      string
	SizeBytes     int64
	Width, Height int
	// Format is the decoded image format ("png", "jpeg", ...), or "" if unknown.
	Format string
}

// Inspect reads a library file's size and image dimensions. Only the image
// header is read.
func Inspect(filename string) (*Info, error) {
	path, err := PathFromURL(mediaFilesPrefix + filename)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	info := &Info{Filename: filename, SizeBytes: st.Size()}
	if cfg, format, err := image.DecodeConfig(f); err == nil {
		info.Width, info.Height, info.Format = cfg.Width, cfg.Height, format
	}
	return info, nil
}
