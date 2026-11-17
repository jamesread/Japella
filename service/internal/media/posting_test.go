package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// photo is a noisy full-colour image, which compresses like a photograph.
func photo(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(2))
	for y := range h {
		for x := range w {
			i := img.PixOffset(x, y)
			img.Pix[i] = uint8(x*255/w) ^ uint8(rng.Intn(64))
			img.Pix[i+1] = uint8(y*255/h) ^ uint8(rng.Intn(64))
			img.Pix[i+2] = uint8(rng.Intn(256))
			img.Pix[i+3] = 0xff
		}
	}
	return img
}

func TestOptimiseForPostingPhoto(t *testing.T) {
	dir := inMediaDir(t)
	writePNG(t, filepath.Join(dir, "big.png"), photo(3000, 1500))

	target := PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000}
	res, err := OptimiseForPosting("big.png", target)
	if err != nil {
		t.Fatal(err)
	}
	if res.Filename != "big.posting.jpg" || res.OptimisedBytes > target.MaxBytes {
		t.Fatalf("got %+v", res)
	}
	out := decodeFile(t, filepath.Join(dir, res.Filename))
	if b := out.Bounds(); max(b.Dx(), b.Dy()) > 2000 {
		t.Fatalf("not resized: %v", b)
	}

	items, _ := List()
	for _, it := range items {
		if it.Filename == "big.png" && it.PostingVersion != "big.posting.jpg" {
			t.Fatalf("original not linked: %+v", it)
		}
		if it.Filename == "big.posting.jpg" && (it.OptimisedOf != "big.png" || it.OptimisedKind != KindPosting) {
			t.Fatalf("copy not linked: %+v", it)
		}
	}

	if _, err := OptimiseForPosting("big.posting.jpg", target); err == nil {
		t.Fatal("preparing a copy should fail")
	}
}

func TestOptimiseForPostingKeepsGraphicsAsPNG(t *testing.T) {
	dir := inMediaDir(t)
	img := image.NewNRGBA(image.Rect(0, 0, 2400, 400))
	for y := range 400 {
		for x := range 2400 {
			if (x/40+y/40)%2 == 0 {
				img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0})
			} else {
				img.SetNRGBA(x, y, color.NRGBA{200, 30, 30, 255})
			}
		}
	}
	writePNG(t, filepath.Join(dir, "logo.png"), img)

	res, err := OptimiseForPosting("logo.png", PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000})
	if err != nil || res.Filename != "logo.posting.png" {
		t.Fatalf("got %+v, %v", res, err)
	}
	if b := decodeFile(t, filepath.Join(dir, res.Filename)).Bounds(); b.Dx() != 2000 {
		t.Fatalf("got %v", b)
	}
}

func TestOptimiseForPostingAppliesOrientation(t *testing.T) {
	dir := inMediaDir(t)
	var enc bytes.Buffer
	if err := jpeg.Encode(&enc, photo(400, 200), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	plain := enc.Bytes()
	rotated := append(append([]byte{0xFF, 0xD8}, orientationEXIF(6)...), plain[2:]...)
	if err := os.WriteFile(filepath.Join(dir, "side.jpg"), rotated, 0644); err != nil {
		t.Fatal(err)
	}

	res, err := OptimiseForPosting("side.jpg", PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000})
	if err != nil || res.Filename == "" {
		t.Fatalf("got %+v, %v", res, err)
	}
	if b := decodeFile(t, filepath.Join(dir, res.Filename)).Bounds(); b.Dx() != 200 || b.Dy() != 400 {
		t.Fatalf("orientation not applied: %v", b)
	}
}

func TestOptimiseForPostingAlreadyFits(t *testing.T) {
	dir := inMediaDir(t)
	var enc bytes.Buffer
	if err := jpeg.Encode(&enc, photo(300, 200), &jpeg.Options{Quality: 60}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "small.jpg"), enc.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := OptimiseForPosting("small.jpg", PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000})
	if err != nil || res.Filename != "" {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func pngChunkBytes(typ string, data []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	out = append(out, typ...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(typ), data...)))
}

func TestOptimiseForPostingRejects(t *testing.T) {
	dir := inMediaDir(t)

	var gifBuf bytes.Buffer
	if err := gif.Encode(&gifBuf, image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White}), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "anim.gif"), gifBuf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, photo(8, 8)); err != nil {
		t.Fatal(err)
	}
	apng, err := insertPNGChunks(pngBuf.Bytes(), [][]byte{pngChunkBytes("acTL", make([]byte, 8))})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "anim.png"), apng, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.png"), []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}

	for file, want := range map[string]string{
		"anim.gif":    "GIFs can't be prepared for posting without losing their animation",
		"anim.png":    "animated PNGs can't be prepared for posting",
		"notes.png":   "failed to read image",
		"missing.png": "no such file",
	} {
		_, err := OptimiseForPosting(file, DefaultPostingTarget)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: got %v, want %q", file, err, want)
		}
	}
}

func TestOptimiseForPostingReplacesCopyInOtherFormat(t *testing.T) {
	dir := inMediaDir(t)
	writePNG(t, filepath.Join(dir, "logo.png"), photo(3000, 1500))
	stale := filepath.Join(dir, "logo.posting.png")
	if err := os.WriteFile(stale, []byte("old copy"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := OptimiseForPosting("logo.png", PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if res.Filename != "logo.posting.jpg" {
		t.Fatalf("got %+v", res)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale copy in the other format was kept: %v", err)
	}
}

func TestOptimiseForPostingNote(t *testing.T) {
	dir := inMediaDir(t)
	writePNG(t, filepath.Join(dir, "big.png"), photo(3000, 1500))

	res, err := OptimiseForPosting("big.png", PostingTarget{MaxBytes: 1000, MaxSide: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Note, "Saved as JPEG at quality ") ||
		!strings.Contains(res.Note, ", resized from 3000×1500 to ") ||
		!strings.HasSuffix(res.Note, "; still over the "+FormatBytesShort(1000)+" limit.") {
		t.Fatalf("note: %q", res.Note)
	}
}
