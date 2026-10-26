package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// inMediaDir runs the test with an empty media library.
func inMediaDir(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	dir, err := GetDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	var buf bytes.Buffer
	// Uncompressed, so recompression has something to gain.
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

func decodeFile(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestOptimiseNames(t *testing.T) {
	if got := OptimisedName("abc.png"); got != "abc.optimised.png" {
		t.Fatal(got)
	}
	if stem, kind := DerivedOf("abc.optimised.png"); stem != "abc" || kind != KindLossless {
		t.Fatal(stem, kind)
	}
	if stem, kind := DerivedOf("abc.posting.jpg"); stem != "abc" || kind != KindPosting {
		t.Fatal(stem, kind)
	}
	if _, kind := DerivedOf("abc.png"); kind != "" {
		t.Fatal(kind)
	}
}

func TestOptimisePNGIsLossless(t *testing.T) {
	dir := inMediaDir(t)

	// Few colours with transparency: becomes a palette image.
	img := image.NewNRGBA(image.Rect(0, 0, 64, 48))
	colours := []color.NRGBA{{255, 0, 0, 255}, {0, 0, 255, 128}, {0, 0, 0, 0}, {10, 200, 30, 255}}
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, colours[(x/8+y/8)%len(colours)])
		}
	}
	writePNG(t, filepath.Join(dir, "a.png"), img)

	res, err := Optimise("a.png")
	if err != nil {
		t.Fatal(err)
	}
	if res.Filename != "a.optimised.png" || res.OptimisedBytes >= res.OriginalBytes {
		t.Fatalf("got %+v", res)
	}
	if !samePixels(img, decodeFile(t, filepath.Join(dir, res.Filename))) {
		t.Fatal("optimised PNG differs")
	}

	items, _ := List()
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Filename] = it
	}
	if byName["a.png"].OptimisedVersion != "a.optimised.png" || byName["a.optimised.png"].OptimisedOf != "a.png" {
		t.Fatalf("links: %+v", byName)
	}

	if _, err := Optimise("a.optimised.png"); err == nil {
		t.Fatal("optimising a copy should fail")
	}
}

func TestOptimisePNGGreyAndNoGain(t *testing.T) {
	dir := inMediaDir(t)

	grey := image.NewRGBA(image.Rect(0, 0, 40, 40))
	rng := rand.New(rand.NewSource(1))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			v := uint8(rng.Intn(256))
			grey.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	writePNG(t, filepath.Join(dir, "g.png"), grey)
	res, err := Optimise("g.png")
	if err != nil || res.Filename == "" {
		t.Fatalf("got %+v, %v", res, err)
	}
	if !samePixels(grey, decodeFile(t, filepath.Join(dir, res.Filename))) {
		t.Fatal("greyscale copy differs")
	}

	// Random colour noise can't be made smaller than a best-compressed file.
	noise := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	rng.Read(noise.Pix)
	var buf bytes.Buffer
	(&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, noise)
	os.WriteFile(filepath.Join(dir, "n.png"), buf.Bytes(), 0644)
	res, err = Optimise("n.png")
	if err != nil || res.Filename != "" {
		t.Fatalf("got %+v, %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "n.optimised.png")); !os.IsNotExist(err) {
		t.Fatal("no copy should be written when nothing is saved")
	}
}

func TestOptimiseJPEGStripsMetadataKeepsOrientation(t *testing.T) {
	dir := inMediaDir(t)

	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for i := range img.Pix {
		img.Pix[i] = uint8(i)
	}
	var enc bytes.Buffer
	if err := jpeg.Encode(&enc, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := enc.Bytes()

	// EXIF with orientation 6, plus a large comment.
	exif := orientationEXIF(6)
	comment := append([]byte{0xFF, 0xFE, 0x10, 0x02}, bytes.Repeat([]byte("x"), 0x1000)...)
	withMeta := append(append(append([]byte{0xFF, 0xD8}, exif...), comment...), plain[2:]...)
	os.WriteFile(filepath.Join(dir, "p.jpg"), withMeta, 0644)

	res, err := Optimise("p.jpg")
	if err != nil || res.Filename == "" {
		t.Fatalf("got %+v, %v", res, err)
	}
	out, _ := os.ReadFile(filepath.Join(dir, res.Filename))
	if bytes.Contains(out, []byte("xxxx")) {
		t.Fatal("comment not removed")
	}
	if !bytes.Contains(out, orientationEXIF(6)) {
		t.Fatal("orientation lost")
	}
	if !samePixels(decodeFile(t, filepath.Join(dir, "p.jpg")), decodeFile(t, filepath.Join(dir, res.Filename))) {
		t.Fatal("JPEG pixels changed")
	}
}

func TestOptimiseRejectsOtherFormats(t *testing.T) {
	dir := inMediaDir(t)
	os.WriteFile(filepath.Join(dir, "a.gif"), []byte("GIF89a...."), 0644)
	if _, err := Optimise("a.gif"); err != ErrNotOptimisable {
		t.Fatalf("got %v", err)
	}
	if _, err := Optimise("../a.gif"); err == nil {
		t.Fatal("path traversal should fail")
	}
}
