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
		if before, ok := strings.CutSuffix(base, "."+k); ok {
			return before, k
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

func optimiseBytes(data []byte) ([]byte, string, error) {
	switch {
	case bytes.HasPrefix(data, pngSignature):
		return optimisePNG(data)
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		return optimiseJPEG(data)
	default:
		return nil, "", ErrNotOptimisable
	}
}

func alreadySmallEnough(data, out []byte, note string) *OptimiseResult {
	// A copy that saves under 1% isn't worth a second library item.
	if len(out) < len(data)-len(data)/100 {
		return nil
	}
	res := &OptimiseResult{
		OriginalBytes:  int64(len(data)),
		OptimisedBytes: int64(len(data)),
		Note:           "This image is already as small as it can be made without losing quality.",
	}
	if note == fullColourNote {
		res.Note += " " + photoNote
	}
	return res
}

func writeOptimisedCopy(dir, filename string, data, out []byte, note string) (*OptimiseResult, error) {
	name := OptimisedName(filename)
	if err := writeAtomic(filepath.Join(dir, name), out); err != nil {
		return nil, err
	}
	return &OptimiseResult{
		Filename:       name,
		OriginalBytes:  int64(len(data)),
		OptimisedBytes: int64(len(out)),
		Note:           note,
	}, nil
}

func optimiseFile(path, filename string) (*OptimiseResult, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path resolved by PathFromURL, confined to the media directory
	if err != nil {
		return nil, err
	}
	out, note, err := optimiseBytes(data)
	if err != nil {
		return nil, err
	}
	if res := alreadySmallEnough(data, out, note); res != nil {
		return res, nil
	}
	return writeOptimisedCopy(filepath.Dir(path), filename, data, out, note)
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
	return optimiseFile(path, filename)
}

func writeAtomic(dst string, data []byte) error {
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { //nolint:gosec // tmp is derived from a PathFromURL path in the media directory
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
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

func colourChunksToKeep(chunks []pngChunk) ([][]byte, error) {
	var keep [][]byte
	for _, c := range chunks {
		if c.typ == "acTL" {
			// Re-encoding would keep only the first frame.
			return nil, fmt.Errorf("animated PNGs can't be optimised")
		}
		if pngColourChunks[c.typ] {
			keep = append(keep, c.raw)
		}
	}
	return keep, nil
}

func pngOptimiseNote(how string) string {
	if how == "" {
		return fullColourNote
	}
	return "Recompressed" + how + "; every pixel is identical to the original."
}

func encodeReducedPNG(img image.Image, keep [][]byte) ([]byte, string, error) {
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
	return out, how, nil
}

func verifyPNGMatch(original image.Image, out []byte) error {
	check, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return fmt.Errorf("optimised PNG didn't match the original; nothing was saved")
	}
	if !samePixels(original, check) {
		return fmt.Errorf("optimised PNG didn't match the original; nothing was saved")
	}
	return nil
}

func preparePNG(data []byte) (img image.Image, keep [][]byte, err error) {
	chunks, err := pngChunks(data)
	if err != nil {
		return nil, nil, err
	}
	keep, err = colourChunksToKeep(chunks)
	if err != nil {
		return nil, nil, err
	}
	img, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read PNG: %w", err)
	}
	return img, keep, nil
}

func optimisePNG(data []byte) ([]byte, string, error) {
	img, keep, err := preparePNG(data)
	if err != nil {
		return nil, "", err
	}
	out, how, err := encodeReducedPNG(img, keep)
	if err != nil {
		return nil, "", err
	}
	if err := verifyPNGMatch(img, out); err != nil {
		return nil, "", err
	}
	return out, pngOptimiseNote(how), nil
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

type pngScan struct {
	colours map[color.NRGBA]bool
	grey    bool
	opaque  bool
	at      func(x, y int) color.NRGBA
	bounds  image.Rectangle
}

// recordPixel updates the scan; returns false when a full-colour image can stop early.
func recordPixel(s *pngScan, c color.NRGBA) bool {
	if c.A != 0xff {
		s.opaque = false
	}
	if c.R != c.G || c.G != c.B {
		s.grey = false
	}
	if len(s.colours) <= 256 {
		s.colours[c] = true
		return true
	}
	return s.grey // stop early when neither reduction applies
}

func scanPNGPixels(img image.Image) pngScan {
	b := img.Bounds()
	at := nrgbaFunc(img)
	s := pngScan{colours: map[color.NRGBA]bool{}, grey: true, opaque: true, at: at, bounds: b}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !recordPixel(&s, at(x, y)) {
				return s
			}
		}
	}
	return s
}

func asGreyscale(s pngScan) *image.Gray {
	g := image.NewGray(s.bounds)
	for y := s.bounds.Min.Y; y < s.bounds.Max.Y; y++ {
		for x := s.bounds.Min.X; x < s.bounds.Max.X; x++ {
			g.SetGray(x, y, color.Gray{Y: s.at(x, y).R})
		}
	}
	return g
}

func paletteLess(a, b color.NRGBA) bool {
	if a.A != b.A {
		return a.A < b.A
	}
	return uint32(a.R)<<16|uint32(a.G)<<8|uint32(a.B) < uint32(b.R)<<16|uint32(b.G)<<8|uint32(b.B)
}

func asPaletted(s pngScan) *image.Paletted {
	palette := make(color.Palette, 0, len(s.colours))
	for c := range s.colours {
		palette = append(palette, c)
	}
	// Transparent entries first keeps the tRNS chunk short.
	sort.Slice(palette, func(i, j int) bool {
		return paletteLess(palette[i].(color.NRGBA), palette[j].(color.NRGBA))
	})
	index := make(map[color.NRGBA]uint8, len(palette))
	for i, c := range palette {
		index[c.(color.NRGBA)] = uint8(i) //nolint:gosec // palette length ≤ 256
	}
	p := image.NewPaletted(s.bounds, palette)
	for y := s.bounds.Min.Y; y < s.bounds.Max.Y; y++ {
		for x := s.bounds.Min.X; x < s.bounds.Max.X; x++ {
			p.SetColorIndex(x, y, index[s.at(x, y)])
		}
	}
	return p
}

// reducePNG returns the smallest lossless representation of an 8-bit image:
// greyscale, or a palette of up to 256 colours.
func reducePNG(img image.Image) (image.Image, string) {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
	default:
		return img, "" // already paletted or greyscale, or 16-bit
	}

	s := scanPNGPixels(img)
	if s.grey && s.opaque {
		return asGreyscale(s), " as greyscale"
	}
	if len(s.colours) <= 256 {
		p := asPaletted(s)
		return p, fmt.Sprintf(" with a %d-colour palette", len(p.Palette))
	}
	return img, ""
}

func sameRGBAPix(a, b image.Image) (same bool, ok bool) {
	ra, okA := a.(*image.RGBA)
	rb, okB := b.(*image.RGBA)
	if !okA || !okB || ra.Stride != rb.Stride {
		return false, false
	}
	return bytes.Equal(ra.Pix, rb.Pix), true
}

func sameNRGBAPix(a, b image.Image) (same bool, ok bool) {
	na, okA := a.(*image.NRGBA)
	nb, okB := b.(*image.NRGBA)
	if !okA || !okB || na.Stride != nb.Stride {
		return false, false
	}
	return bytes.Equal(na.Pix, nb.Pix), true
}

func samePixBytes(a, b image.Image) (same bool, ok bool) {
	if same, ok := sameRGBAPix(a, b); ok {
		return same, true
	}
	return sameNRGBAPix(a, b)
}

func sameNRGBAPixels(a, b image.Image) bool {
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

func sameNRGBA64Pixels(a, b image.Image) bool {
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

func samePixels(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	// Unchanged representation (the common case for photos): compare bytes.
	if same, ok := samePixBytes(a, b); ok {
		return same
	}
	if is8Bit(a) && is8Bit(b) {
		return sameNRGBAPixels(a, b)
	}
	return sameNRGBA64Pixels(a, b)
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

func jpegMarkerAt(data []byte, p int) (byte, error) {
	if p+4 > len(data) {
		return 0, fmt.Errorf("unexpected JPEG layout")
	}
	if data[p] != 0xFF {
		return 0, fmt.Errorf("unexpected JPEG layout")
	}
	return data[p+1], nil
}

func jpegSegmentEnd(data []byte, p int) (int, error) {
	n := int(binary.BigEndian.Uint16(data[p+2:]))
	end := p + 2 + n
	if n < 2 {
		return 0, fmt.Errorf("truncated JPEG")
	}
	if end > len(data) {
		return 0, fmt.Errorf("truncated JPEG")
	}
	return end, nil
}

func jpegSegmentBounds(data []byte, p int) (marker byte, end int, err error) {
	marker, err = jpegMarkerAt(data, p)
	if err != nil {
		return 0, 0, err
	}
	if marker == 0xFF { // fill byte
		return marker, p + 1, nil
	}
	end, err = jpegSegmentEnd(data, p)
	return marker, end, err
}

func applyJPEGSegment(out []byte, data []byte, p, end int, marker byte, orientation int) ([]byte, int, bool) {
	if marker == 0xDA { // start of scan: the image data follows, copied untouched
		return append(out, data[p:]...), orientation, true
	}
	seg, payload := data[p:end], data[p+4:end]
	if keepJPEGSegment(marker, payload) {
		return append(out, seg...), orientation, false
	}
	if marker == 0xE1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
		orientation = exifOrientation(payload[6:])
	}
	return out, orientation, false
}

func stripJPEGMetadata(data []byte) ([]byte, int, error) {
	out := []byte{0xFF, 0xD8}
	orientation := 0
	for p := 2; ; {
		marker, end, err := jpegSegmentBounds(data, p)
		if err != nil {
			return nil, 0, err
		}
		if marker == 0xFF { // fill byte
			p = end
			continue
		}
		var done bool
		out, orientation, done = applyJPEGSegment(out, data, p, end, marker, orientation)
		if done {
			return out, orientation, nil
		}
		p = end
	}
}

func withJPEGOrientation(out []byte, orientation int) []byte {
	if orientation <= 1 {
		return out
	}
	// Keep only the rotation, so the image isn't shown sideways.
	return append(out[:2], append(orientationEXIF(orientation), out[2:]...)...)
}

func jpegDimsMatch(cfg image.Config, out []byte) bool {
	check, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		return false
	}
	return check.Width == cfg.Width && check.Height == cfg.Height
}

func optimiseJPEG(data []byte) ([]byte, string, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read JPEG: %w", err)
	}
	out, orientation, err := stripJPEGMetadata(data)
	if err != nil {
		return nil, "", err
	}
	out = withJPEGOrientation(out, orientation)
	if !jpegDimsMatch(cfg, out) {
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

func tiffByteOrder(tiff []byte) binary.ByteOrder {
	switch string(tiff[:2]) {
	case "II":
		return binary.LittleEndian
	case "MM":
		return binary.BigEndian
	default:
		return nil
	}
}

func isOrientationTag(tiff []byte, bo binary.ByteOrder, e int) bool {
	return bo.Uint16(tiff[e:]) == 0x0112 && bo.Uint16(tiff[e+2:]) == 3
}

func validOrientation(v int) bool {
	return v >= 1 && v <= 8
}

func orientationTagValue(tiff []byte, bo binary.ByteOrder, e int) (int, bool) {
	if e+12 > len(tiff) {
		return 0, true // stop
	}
	if !isOrientationTag(tiff, bo, e) {
		return 0, false
	}
	v := int(bo.Uint16(tiff[e+8:]))
	if validOrientation(v) {
		return v, true
	}
	return 0, true
}

func orientationFromIFD(tiff []byte, bo binary.ByteOrder, ifd int) int {
	count := int(bo.Uint16(tiff[ifd:]))
	for i := range count {
		v, done := orientationTagValue(tiff, bo, ifd+2+i*12)
		if done {
			return v
		}
	}
	return 0
}

// exifOrientation reads the Orientation tag from a TIFF-format EXIF block, or 0.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 0
	}
	bo := tiffByteOrder(tiff)
	if bo == nil {
		return 0
	}
	ifd := int(bo.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	return orientationFromIFD(tiff, bo, ifd)
}

// orientationEXIF is an APP1 segment holding only an Orientation tag.
func orientationEXIF(orientation int) []byte {
	if orientation < 1 || orientation > 8 {
		orientation = 1
	}
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2)) //nolint:gosec // payload is a fixed 32-byte segment
	return append(seg, payload...)
}
