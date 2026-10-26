package media

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestInspect(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	mediaDir, err := GetDir()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(mediaDir, "pic.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	f.Close()

	info, err := Inspect("pic.png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 30 || info.Height != 20 || info.Format != "png" || info.SizeBytes == 0 {
		t.Fatalf("got %+v", info)
	}

	for _, bad := range []string{"../pic.png", "missing.png", ""} {
		if _, err := Inspect(bad); err == nil {
			t.Errorf("Inspect(%q) should fail", bad)
		}
	}
}
