// Package medialimits checks whether a media file suits each social
// network's upload limits: format, file size, dimensions and shape.
//
// The limits are the networks' published defaults; Mastodon instances can
// configure their own.
package medialimits

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// Levels, worst last.
const (
	LevelOK      = "ok"
	LevelWarning = "warning"
	LevelError   = "error"
)

// Limits are one network's rules for attached media. Zero means no limit.
type Limits struct {
	Name string

	ImageFormats  []string // lowercase extensions without '.'
	ImageMaxBytes int64
	// GIFMaxBytes applies to GIFs instead of ImageMaxBytes when set.
	GIFMaxBytes int64
	// StaticGIFs: GIFs are accepted but shown without animation.
	StaticGIFs bool

	VideoFormats  []string
	VideoMaxBytes int64

	// Hard dimension limits: larger images are rejected.
	MinWidth, MinHeight int
	MaxWidth, MaxHeight int
	// MaxPixels: larger images are downscaled by the network.
	MaxPixels int64
	// ResizeLongestSide: images longer than this are shown downscaled.
	ResizeLongestSide int
	// Images more elongated than this (long side / short side) are cropped
	// in timelines.
	CropAspectRatio float64
}

var byProtocol = map[string]Limits{
	"mastodon": {
		Name:          "Mastodon",
		ImageFormats:  []string{"jpg", "jpeg", "png", "gif", "webp", "heic", "heif", "avif"},
		ImageMaxBytes: 16 << 20,
		VideoFormats:  []string{"mp4", "m4v", "mov", "webm"},
		VideoMaxBytes: 99 << 20,
		MaxPixels:     7680 * 4320,
	},
	"bluesky": {
		Name:              "Bluesky",
		ImageFormats:      []string{"jpg", "jpeg", "png", "webp", "gif"},
		ImageMaxBytes:     1_000_000,
		VideoFormats:      []string{"mp4"},
		VideoMaxBytes:     100_000_000,
		StaticGIFs:        true,
		ResizeLongestSide: 2000,
		CropAspectRatio:   3,
	},
	"x": {
		Name:            "X",
		ImageFormats:    []string{"jpg", "jpeg", "png", "gif", "webp"},
		ImageMaxBytes:   5 << 20,
		GIFMaxBytes:     15 << 20,
		VideoFormats:    []string{"mp4", "mov"},
		VideoMaxBytes:   512 << 20,
		MinWidth:        4,
		MinHeight:       4,
		MaxWidth:        8192,
		MaxHeight:       8192,
		CropAspectRatio: 3,
	},
}

// ForProtocol returns a connector protocol's limits, if Japella knows them.
func ForProtocol(protocol string) (Limits, bool) {
	l, ok := byProtocol[strings.ToLower(protocol)]
	return l, ok
}

// Media describes a file. Width and Height are 0 when unknown (videos, or
// formats Japella can't read).
type Media struct {
	Filename      string
	SizeBytes     int64
	Width, Height int
}

// Check is one finding about a file.
type Check struct {
	Level   string
	Message string
}

// Ext returns the file's lowercase extension without '.'.
func (m Media) Ext() string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(m.Filename)), ".")
}

// IsVideo reports whether the file's extension is a video format.
func IsVideo(ext string) bool {
	return slices.Contains([]string{"mp4", "m4v", "mov", "webm", "ogg", "avi", "mkv"}, ext)
}

// FormatBytes renders a byte count like the UI: "512 bytes", "1.5 KB", "2.3 MB".
func FormatBytes(n int64) string {
	switch {
	case n < 1e3:
		return fmt.Sprintf("%d bytes", n)
	case n < 1e6:
		return fmt.Sprintf("%.1f KB", float64(n)/1e3)
	case n < 1e9:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/1e9)
	}
}

// Evaluate checks a file against a network's limits. The verdict is the
// worst level among the checks.
func Evaluate(l Limits, m Media) (verdict string, checks []Check) {
	add := func(level, format string, args ...any) {
		checks = append(checks, Check{Level: level, Message: fmt.Sprintf(format, args...)})
	}

	ext := m.Ext()
	video := IsVideo(ext)

	formats, maxBytes, kind := l.ImageFormats, l.ImageMaxBytes, "image"
	if video {
		formats, maxBytes, kind = l.VideoFormats, l.VideoMaxBytes, "video"
	} else if ext == "gif" && l.GIFMaxBytes > 0 {
		maxBytes, kind = l.GIFMaxBytes, "GIF"
	}

	if !slices.Contains(formats, ext) {
		add(LevelError, "%s doesn't accept .%s files", l.Name, ext)
	} else if ext == "gif" && l.StaticGIFs {
		add(LevelWarning, "%s shows GIFs without animation", l.Name)
	}

	switch {
	case maxBytes == 0:
	case m.SizeBytes > maxBytes:
		add(LevelError, "%s is over the %s %s limit", FormatBytes(m.SizeBytes), FormatBytes(maxBytes), kind)
	case m.SizeBytes > maxBytes*9/10:
		add(LevelWarning, "%s is close to the %s %s limit", FormatBytes(m.SizeBytes), FormatBytes(maxBytes), kind)
	default:
		add(LevelOK, "%s is within the %s %s limit", FormatBytes(m.SizeBytes), FormatBytes(maxBytes), kind)
	}

	if video || m.Width == 0 || m.Height == 0 {
		return worst(checks), checks
	}

	w, h := m.Width, m.Height
	dimsOK := true
	if (l.MinWidth > 0 && w < l.MinWidth) || (l.MinHeight > 0 && h < l.MinHeight) {
		add(LevelError, "%d×%d is smaller than the %d×%d minimum", w, h, l.MinWidth, l.MinHeight)
		dimsOK = false
	}
	if (l.MaxWidth > 0 && w > l.MaxWidth) || (l.MaxHeight > 0 && h > l.MaxHeight) {
		add(LevelError, "%d×%d is larger than the %d×%d maximum", w, h, l.MaxWidth, l.MaxHeight)
		dimsOK = false
	}
	if l.MaxPixels > 0 && int64(w)*int64(h) > l.MaxPixels {
		add(LevelWarning, "%d×%d is over %.1f megapixels and will be downscaled", w, h, float64(l.MaxPixels)/1e6)
		dimsOK = false
	}
	if l.ResizeLongestSide > 0 && max(w, h) > l.ResizeLongestSide {
		add(LevelWarning, "%d×%d will be shown downscaled to %dpx on the longest side", w, h, l.ResizeLongestSide)
		dimsOK = false
	}
	if l.CropAspectRatio > 0 {
		if ratio := float64(max(w, h)) / float64(min(w, h)); ratio > l.CropAspectRatio {
			add(LevelWarning, "%d×%d is very %s (%.1f:1) and will be cropped in timelines", w, h, shape(w, h), ratio)
			dimsOK = false
		}
	}
	if dimsOK {
		add(LevelOK, "%d×%d dimensions are fine", w, h)
	}

	return worst(checks), checks
}

func shape(w, h int) string {
	if w > h {
		return "wide"
	}
	return "tall"
}

func worst(checks []Check) string {
	verdict := LevelOK
	for _, c := range checks {
		if c.Level == LevelError {
			return LevelError
		}
		if c.Level == LevelWarning {
			verdict = LevelWarning
		}
	}
	return verdict
}
