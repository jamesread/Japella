package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Optimised copies sit beside their original as "<stem>.<kind>.<ext>", so
// they are ordinary library items and the link to the original needs no
// database. A posting copy may change format, so originals are matched by stem.
const (
	KindLossless = "optimised"
	KindPosting  = "posting"
)

// OptimisedName returns the filename of an original's lossless copy.
func OptimisedName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext) + "." + KindLossless + ext
}

// postingName returns the filename of an original's posting copy in format ext.
func postingName(filename, ext string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename)) + "." + KindPosting + ext
}

// DerivedOf splits an optimised copy's name into its original's stem and
// the kind of copy. Both are "" for originals.
func DerivedOf(filename string) (stem, kind string) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, k := range []string{KindLossless, KindPosting} {
		if strings.HasSuffix(base, "."+k) {
			return strings.TrimSuffix(base, "."+k), k
		}
	}
	return "", ""
}

// Stem is a filename without its extension.
func Stem(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// ErrNotOptimisable is returned for files Japella can't optimise losslessly.
var ErrNotOptimisable = errors.New("only PNG and JPEG images can be optimised")

// OptimiseResult describes one optimisation.
type OptimiseResult struct {
	Filename       string // the optimised copy; "" when none was written
	OriginalBytes  int64
	OptimisedBytes int64
	// Note explains a result with no copy, or what was changed.
	Note string
}

// Optimise writes a smaller copy of a library image that looks identical:
// PNGs are recompressed (with fewer colour channels or a palette when that
// loses nothing) and checked pixel for pixel; JPEGs keep their image data
// untouched and lose only metadata such as camera details and location. The
// original is never changed. No copy is written if it wouldn't be smaller.
func Optimise(filename string) (*OptimiseResult, error) {
	if _, kind := DerivedOf(filename); kind != "" {
		return nil, fmt.Errorf("%s is already an optimised copy", filename)
	}
	path, err := PathFromURL(mediaFilesPrefix + filename)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var out []byte
	var note string
	switch {
	case bytes.HasPrefix(data, pngSignature):
		out, note, err = optimisePNG(data)
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		out, note, err = optimiseJPEG(data)
	default:
		return nil, ErrNotOptimisable
	}
	if err != nil {
		return nil, err
	}

	res := &OptimiseResult{OriginalBytes: int64(len(data))}
	// A copy that saves under 1% isn't worth a second library item.
	if len(out) >= len(data)-len(data)/100 {
		res.OptimisedBytes = res.OriginalBytes
		res.Note = "This image is already as small as it can be made without losing quality."
		if note == fullColourNote {
			res.Note += " " + photoNote
		}
		return res, nil
	}

	name := OptimisedName(filename)
	if err := writeAtomic(filepath.Join(filepath.Dir(path), name), out); err != nil {
		return nil, err
	}
	res.Filename, res.OptimisedBytes, res.Note = name, int64(len(out)), note
	return res, nil
}

func writeAtomic(dst string, data []byte) error {
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// --- PNG ---

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// fullColourNote is the result for full-colour PNGs, which can only be recompressed.
const fullColourNote = "Recompressed; every pixel is identical to the original."

// photoNote explains small savings on full-colour images.
const photoNote = "It looks like a photo: lossless compression can only shrink photos a little; making them much smaller means changing pixels."

type pngChunk struct {
	typ string
	raw []byte // length, type, data and CRC, as in the file
}

func pngChunks(data []byte) ([]pngChunk, error) {
	var chunks []pngChunk
	for p := len(pngSignature); p < len(data); {
		if p+8 > len(data) {
			return nil, fmt.Errorf("truncated PNG")
		}
		n := int(binary.BigEndian.Uint32(data[p:]))
		end := p + 12 + n
		if n < 0 || end > len(data) {
			return nil, fmt.Errorf("truncated PNG")
		}
		chunks = append(chunks, pngChunk{typ: string(data[p+4 : p+8]), raw: data[p:end]})
		p = end
	}
	return chunks, nil
}

// Chunks that change how pixels are displayed; kept so the copy looks the same.
var pngColourChunks = map[string]bool{"iCCP": true, "sRGB": true, "gAMA": true, "cHRM": true, "cICP": true, "pHYs": true}

func optimisePNG(data []byte) ([]byte, string, error) {
	chunks, err := pngChunks(data)
	if err != nil {
		return nil, "", err
	}
	var keep [][]byte
	for _, c := range chunks {
		if c.typ == "acTL" {
			// Re-encoding would keep only the first frame.
			return nil, "", fmt.Errorf("animated PNGs can't be optimised")
		}
		if pngColourChunks[c.typ] {
			keep = append(keep, c.raw)
		}
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read PNG: %w", err)
	}

	reduced, how := reducePNG(img)
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: pngCompressionLevel(reduced)}
	if err := enc.Encode(&buf, reduced); err != nil {
		return nil, "", err
	}
	out, err := insertPNGChunks(buf.Bytes(), keep)
	if err != nil {
		return nil, "", err
	}

	// Never hand back a copy that differs from the original.
	check, err := png.Decode(bytes.NewReader(out))
	if err != nil || !samePixels(img, check) {
		return nil, "", fmt.Errorf("optimised PNG didn't match the original; nothing was saved")
	}
	if how == "" {
		return out, fullColourNote, nil
	}
	return out, "Recompressed" + how + "; every pixel is identical to the original.", nil
}

// Maximum compression takes ~7x longer than the default; on large full-colour
// images (photos) that's ~20s to save ~5%, so it's kept for images where it pays.
const bestCompressionMaxPixels = 4_000_000

func pngCompressionLevel(img image.Image) png.CompressionLevel {
	switch img.(type) {
	case *image.Paletted, *image.Gray:
		return png.BestCompression
	}
	if b := img.Bounds(); b.Dx()*b.Dy() <= bestCompressionMaxPixels {
		return png.BestCompression
	}
	return png.DefaultCompression
}

// nrgbaFunc reads 8-bit pixels without the per-pixel interface calls of
// image.At, which dominate on large images.
func nrgbaFunc(img image.Image) func(x, y int) color.NRGBA {
	switch m := img.(type) {
	case *image.NRGBA:
		return func(x, y int) color.NRGBA {
			i := m.PixOffset(x, y)
			return color.NRGBA{m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3]}
		}
	case *image.RGBA:
		return func(x, y int) color.NRGBA {
			i := m.PixOffset(x, y)
			c := color.RGBA{m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3]}
			if c.A == 0xff {
				return color.NRGBA{c.R, c.G, c.B, 0xff}
			}
			return color.NRGBAModel.Convert(c).(color.NRGBA)
		}
	}
	return func(x, y int) color.NRGBA {
		return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	}
}

// reducePNG returns the smallest lossless representation of an 8-bit image:
// greyscale, or a palette of up to 256 colours.
func reducePNG(img image.Image) (image.Image, string) {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
	default:
		return img, "" // already paletted or greyscale, or 16-bit
	}

	b := img.Bounds()
	at := nrgbaFunc(img)
	colours := map[color.NRGBA]bool{}
	grey, opaque := true, true
scan:
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := at(x, y)
			if c.A != 0xff {
				opaque = false
			}
			if c.R != c.G || c.G != c.B {
				grey = false
			}
			if len(colours) <= 256 {
				colours[c] = true
			} else if !grey {
				break scan // a full-colour image: neither reduction applies
			}
		}
	}

	if grey && opaque {
		g := image.NewGray(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				g.SetGray(x, y, color.Gray{Y: at(x, y).R})
			}
		}
		return g, " as greyscale"
	}

	if len(colours) <= 256 {
		palette := make(color.Palette, 0, len(colours))
		for c := range colours {
			palette = append(palette, c)
		}
		// Transparent entries first keeps the tRNS chunk short.
		sort.Slice(palette, func(i, j int) bool {
			a, b := palette[i].(color.NRGBA), palette[j].(color.NRGBA)
			if a.A != b.A {
				return a.A < b.A
			}
			return uint32(a.R)<<16|uint32(a.G)<<8|uint32(a.B) < uint32(b.R)<<16|uint32(b.G)<<8|uint32(b.B)
		})
		index := make(map[color.NRGBA]uint8, len(palette))
		for i, c := range palette {
			index[c.(color.NRGBA)] = uint8(i)
		}
		p := image.NewPaletted(b, palette)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				p.SetColorIndex(x, y, index[at(x, y)])
			}
		}
		return p, fmt.Sprintf(" with a %d-colour palette", len(palette))
	}
	return img, ""
}

func samePixels(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	// Unchanged representation (the common case for photos): compare bytes.
	if ra, ok := a.(*image.RGBA); ok {
		if rb, ok := b.(*image.RGBA); ok && ra.Stride == rb.Stride {
			return bytes.Equal(ra.Pix, rb.Pix)
		}
	}
	if na, ok := a.(*image.NRGBA); ok {
		if nb, ok := b.(*image.NRGBA); ok && na.Stride == nb.Stride {
			return bytes.Equal(na.Pix, nb.Pix)
		}
	}
	if is8Bit(a) && is8Bit(b) {
		atA, atB := nrgbaFunc(a), nrgbaFunc(b)
		r := a.Bounds()
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if atA(x, y) != atB(x, y) {
					return false
				}
			}
		}
		return true
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if color.NRGBA64Model.Convert(a.At(x, y)) != color.NRGBA64Model.Convert(b.At(x, y)) {
				return false
			}
		}
	}
	return true
}

func is8Bit(img image.Image) bool {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA, *image.Gray, *image.Paletted:
		return true
	}
	return false
}

// insertPNGChunks adds raw chunks straight after IHDR, where colour chunks belong.
func insertPNGChunks(data []byte, chunks [][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return data, nil
	}
	ihdrEnd := len(pngSignature) + 12 + 13
	if len(data) < ihdrEnd || string(data[len(pngSignature)+4:len(pngSignature)+8]) != "IHDR" {
		return nil, fmt.Errorf("unexpected PNG layout")
	}
	out := append([]byte{}, data[:ihdrEnd]...)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return append(out, data[ihdrEnd:]...), nil
}

// --- JPEG ---

func optimiseJPEG(data []byte) ([]byte, string, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read JPEG: %w", err)
	}

	out := []byte{0xFF, 0xD8}
	orientation := 0
	p := 2
	for {
		if p+4 > len(data) || data[p] != 0xFF {
			return nil, "", fmt.Errorf("unexpected JPEG layout")
		}
		marker := data[p+1]
		if marker == 0xFF { // fill byte
			p++
			continue
		}
		n := int(binary.BigEndian.Uint16(data[p+2:]))
		end := p + 2 + n
		if n < 2 || end > len(data) {
			return nil, "", fmt.Errorf("truncated JPEG")
		}
		seg, payload := data[p:end], data[p+4:end]

		if marker == 0xDA { // start of scan: the image data follows, copied untouched
			out = append(out, data[p:]...)
			break
		}
		if keepJPEGSegment(marker, payload) {
			out = append(out, seg...)
		} else if marker == 0xE1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			orientation = exifOrientation(payload[6:])
		}
		p = end
	}

	if orientation > 1 {
		// Keep only the rotation, so the image isn't shown sideways.
		out = append(out[:2], append(orientationEXIF(orientation), out[2:]...)...)
	}

	check, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || check.Width != cfg.Width || check.Height != cfg.Height {
		return nil, "", fmt.Errorf("optimised JPEG didn't match the original; nothing was saved")
	}
	return out, "Removed metadata (camera details, location, embedded thumbnails); the image data is untouched.", nil
}

// keepJPEGSegment keeps everything needed to decode and colour the image:
// JFIF, ICC colour profiles and Adobe colour transforms, plus all non-APP
// segments other than comments.
func keepJPEGSegment(marker byte, payload []byte) bool {
	switch {
	case marker == 0xE0, marker == 0xEE:
		return true
	case marker == 0xE2:
		return bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00"))
	case marker >= 0xE1 && marker <= 0xEF, marker == 0xFE:
		return false
	default:
		return true
	}
}

// exifOrientation reads the Orientation tag from a TIFF-format EXIF block, or 0.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	ifd := int(bo.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	count := int(bo.Uint16(tiff[ifd:]))
	for i := 0; i < count; i++ {
		e := ifd + 2 + i*12
		if e+12 > len(tiff) {
			return 0
		}
		if bo.Uint16(tiff[e:]) == 0x0112 && bo.Uint16(tiff[e+2:]) == 3 {
			return int(bo.Uint16(tiff[e+8:]))
		}
	}
	return 0
}

// orientationEXIF is an APP1 segment holding only an Orientation tag.
func orientationEXIF(orientation int) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}
