package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// photo is a noisy full-colour image, which compresses like a photograph.
func photo(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(2))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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
	for y := 0; y < 400; y++ {
		for x := 0; x < 2400; x++ {
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
	os.WriteFile(filepath.Join(dir, "side.jpg"), rotated, 0644)

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
	jpeg.Encode(&enc, photo(300, 200), &jpeg.Options{Quality: 60})
	os.WriteFile(filepath.Join(dir, "small.jpg"), enc.Bytes(), 0644)

	res, err := OptimiseForPosting("small.jpg", PostingTarget{MaxBytes: 1_000_000, MaxSide: 2000})
	if err != nil || res.Filename != "" {
		t.Fatalf("got %+v, %v", res, err)
	}
}
