package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"

	xdraw "golang.org/x/image/draw"
)

// PostingTarget is what a posting copy must fit: every network's limits at once.
type PostingTarget struct {
	MaxBytes int64
	// MaxSide is the longest side, in pixels, worth keeping.
	MaxSide int
}

// DefaultPostingTarget is used when no network limits are known.
var DefaultPostingTarget = PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000}

// JPEG qualities tried, best first, before shrinking further.
var postingQualities = []int{90, 85, 80, 75}

const (
	// Each extra shrink keeps this share of the longest side.
	postingShrinkStep = 0.85
	postingMaxShrinks = 6
)

// OptimiseForPosting saves a copy of a library image that fits target,
// resized to at most target.MaxSide and re-saved at high quality: JPEG for
// photos, PNG for graphics and transparent images. Unlike Optimise this
// changes pixels, though usually invisibly. The original is never changed.
func OptimiseForPosting(filename string, target PostingTarget) (*OptimiseResult, error) {
	if _, kind := DerivedOf(filename); kind != "" {
		return nil, fmt.Errorf("%s is already an optimised copy", filename)
	}
	path, data, err := readMedia(filename)
	if err != nil {
		return nil, err
	}
	img, err := decodeForPosting(data)
	if err != nil {
		return nil, err
	}
	return preparePostingCopy(filename, path, data, img, target.withDefaults())
}

func (t PostingTarget) withDefaults() PostingTarget {
	if t.MaxBytes <= 0 {
		t.MaxBytes = DefaultPostingTarget.MaxBytes
	}
	if t.MaxSide <= 0 {
		t.MaxSide = DefaultPostingTarget.MaxSide
	}
	return t
}

func readMedia(filename string) (string, []byte, error) {
	path, err := PathFromURL(mediaFilesPrefix + filename)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(path) //nolint:gosec // path resolved by PathFromURL, confined to the media directory
	return path, data, err
}

// decodeForPosting rejects images that re-encoding would spoil.
func decodeForPosting(data []byte) (image.Image, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	switch format {
	case "png":
		return img, rejectAnimatedPNG(data)
	case "jpeg":
		// Pixels are re-encoded without EXIF, so apply its rotation now.
		return applyOrientation(img, jpegOrientation(data)), nil
	case "gif":
		return nil, fmt.Errorf("GIFs can't be prepared for posting without losing their animation")
	}
	return img, nil
}

// rejectAnimatedPNG reports an APNG; a PNG whose chunks can't be read is let
// through, as image.Decode already accepted it.
func rejectAnimatedPNG(data []byte) error {
	chunks, err := pngChunks(data)
	if err != nil {
		return nil //nolint:nilerr // image.Decode accepted it, so treat it as a still PNG
	}
	if slices.ContainsFunc(chunks, func(c pngChunk) bool { return c.typ == "acTL" }) {
		return fmt.Errorf("animated PNGs can't be prepared for posting")
	}
	return nil
}

func preparePostingCopy(filename, path string, data []byte, img image.Image, target PostingTarget) (*OptimiseResult, error) {
	src := toNRGBA(img)
	graphic := !isOpaque(src) || fewColours(src)

	out, size, quality, err := encodeToFit(src, target, graphic)
	if err != nil {
		return nil, err
	}

	res := &OptimiseResult{OriginalBytes: int64(len(data)), OptimisedBytes: int64(len(out))}
	bounds := img.Bounds()
	if wouldNotHelp(out, data, size, bounds) {
		res.OptimisedBytes = res.OriginalBytes
		res.Note = "This image already fits; a posting copy wouldn't be smaller."
		return res, nil
	}

	name, err := savePostingCopy(filename, path, out, graphic)
	if err != nil {
		return nil, err
	}
	res.Filename = name
	res.Note = postingNote(quality, graphic, bounds, size, int64(len(out)), target.MaxBytes)
	return res, nil
}

// wouldNotHelp is true when the copy has no fewer bytes and the same dimensions.
func wouldNotHelp(out, original []byte, size image.Point, bounds image.Rectangle) bool {
	return len(out) >= len(original) && size == bounds.Size()
}

func savePostingCopy(filename, path string, out []byte, graphic bool) (string, error) {
	ext := postingExt(graphic)
	name := postingName(filename, ext)
	dir := filepath.Dir(path)
	removeOtherPostingCopy(dir, filename, ext)
	if err := writeAtomic(filepath.Join(dir, name), out); err != nil {
		return "", err
	}
	return name, nil
}

func postingExt(graphic bool) string {
	if graphic {
		return ".png"
	}
	return ".jpg"
}

// A previous copy may have been saved in the other format.
func removeOtherPostingCopy(dir, filename, ext string) {
	for _, old := range []string{".jpg", ".png"} {
		if old != ext {
			_ = os.Remove(filepath.Join(dir, postingName(filename, old)))
		}
	}
}

func postingNote(quality int, graphic bool, bounds image.Rectangle, size image.Point, outBytes, maxBytes int64) string {
	how := fmt.Sprintf("JPEG at quality %d", quality)
	if graphic {
		how = "PNG (kept lossless for graphics and transparency)"
	}
	note := "Saved as " + how
	if size != bounds.Size() {
		note += fmt.Sprintf(", resized from %d×%d to %d×%d", bounds.Dx(), bounds.Dy(), size.X, size.Y)
	}
	if outBytes > maxBytes {
		note += fmt.Sprintf("; still over the %s limit", FormatBytesShort(maxBytes))
	}
	return note + "."
}

// encodeToFit encodes img no larger than target.MaxSide, lowering JPEG
// quality and then dimensions until it fits target.MaxBytes.
func encodeToFit(img *image.NRGBA, target PostingTarget, graphic bool) ([]byte, image.Point, int, error) {
	b := img.Bounds()
	long := max(b.Dx(), b.Dy())
	side := min(long, target.MaxSide)

	var best []byte
	var bestSize image.Point
	bestQuality := 0
	for shrink := 0; shrink <= postingMaxShrinks; shrink++ {
		scaled := resize(img, side)
		if graphic {
			reduced, _ := reducePNG(scaled)
			var buf bytes.Buffer
			if err := (&png.Encoder{CompressionLevel: pngCompressionLevel(reduced)}).Encode(&buf, reduced); err != nil {
				return nil, image.Point{}, 0, err
			}
			best, bestSize = buf.Bytes(), scaled.Bounds().Size()
			if int64(buf.Len()) <= target.MaxBytes {
				return best, bestSize, 0, nil
			}
		} else {
			for _, q := range postingQualities {
				var buf bytes.Buffer
				if err := jpeg.Encode(&buf, opaqueRGBA(scaled), &jpeg.Options{Quality: q}); err != nil {
					return nil, image.Point{}, 0, err
				}
				best, bestSize, bestQuality = buf.Bytes(), scaled.Bounds().Size(), q
				if int64(buf.Len()) <= target.MaxBytes {
					return best, bestSize, q, nil
				}
			}
		}
		side = int(math.Round(float64(side) * postingShrinkStep))
		if side < 64 {
			break
		}
	}
	// Smallest attempt; the result note says it's still over the limit.
	return best, bestSize, bestQuality, nil
}

// resize scales img so its longest side is at most side.
func resize(img *image.NRGBA, side int) *image.NRGBA {
	b := img.Bounds()
	long := max(b.Dx(), b.Dy())
	if long <= side {
		return img
	}
	scale := float64(side) / float64(long)
	w := max(1, int(math.Round(float64(b.Dx())*scale)))
	h := max(1, int(math.Round(float64(b.Dy())*scale)))
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst
}

// opaqueRGBA views an opaque NRGBA image as RGBA (the bytes are the same),
// which the JPEG encoder handles much faster.
func opaqueRGBA(img *image.NRGBA) *image.RGBA {
	return &image.RGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect}
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok && n.Bounds().Min == (image.Point{}) {
		return n
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	xdraw.Draw(dst, dst.Bounds(), img, b.Min, xdraw.Src)
	return dst
}

func isOpaque(img *image.NRGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0xff {
			return false
		}
	}
	return true
}

// fewColours reports whether img has at most 256 colours: a graphic, which
// JPEG would blur and a palette PNG stores better.
func fewColours(img *image.NRGBA) bool {
	seen := map[color.NRGBA]bool{}
	for i := 0; i+3 < len(img.Pix); i += 4 {
		seen[color.NRGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}] = true
		if len(seen) > 256 {
			return false
		}
	}
	return true
}

// jpegOrientation returns a JPEG's EXIF orientation, or 0.
func jpegOrientation(data []byte) int {
	for p := 2; p+4 <= len(data) && data[p] == 0xFF; {
		marker := data[p+1]
		n := int(data[p+2])<<8 | int(data[p+3])
		end := p + 2 + n
		if marker == 0xDA || n < 2 || end > len(data) {
			return 0
		}
		if payload := data[p+4 : end]; marker == 0xE1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			return exifOrientation(payload[6:])
		}
		p = end
	}
	return 0
}

// applyOrientation rotates/flips img upright for an EXIF orientation (2-8).
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return img
	}
	src := toNRGBA(img)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			si, di := src.PixOffset(x, y), dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// FormatBytesShort renders a limit like "1 MB" or "5.2 MB".
func FormatBytesShort(n int64) string {
	s := fmt.Sprintf("%.1f MB", float64(n)/1e6)
	if n%1_000_000 == 0 {
		s = fmt.Sprintf("%d MB", n/1_000_000)
	}
	return s
}
