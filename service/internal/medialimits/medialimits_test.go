package medialimits

import (
	"strings"
	"testing"
)

func evaluate(t *testing.T, protocol string, m Media) (string, []Check) {
	t.Helper()
	l, ok := ForProtocol(protocol)
	if !ok {
		t.Fatalf("no limits for %s", protocol)
	}
	return Evaluate(l, m)
}

func hasCheck(checks []Check, level, contains string) bool {
	for _, c := range checks {
		if c.Level == level && strings.Contains(c.Message, contains) {
			return true
		}
	}
	return false
}

func TestEvaluateFitsEverywhere(t *testing.T) {
	m := Media{Filename: "photo.jpg", SizeBytes: 300_000, Width: 1600, Height: 900}
	for _, p := range []string{"mastodon", "bluesky", "x"} {
		if verdict, checks := evaluate(t, p, m); verdict != LevelOK {
			t.Errorf("%s: verdict %s, %+v", p, verdict, checks)
		}
	}
}

func TestEvaluateSize(t *testing.T) {
	verdict, checks := evaluate(t, "bluesky", Media{Filename: "big.png", SizeBytes: 2_500_000, Width: 1000, Height: 1000})
	if verdict != LevelError || !hasCheck(checks, LevelError, "over the 1.0 MB image limit") {
		t.Fatalf("got %s %+v", verdict, checks)
	}

	verdict, checks = evaluate(t, "bluesky", Media{Filename: "near.png", SizeBytes: 950_000, Width: 1000, Height: 1000})
	if verdict != LevelWarning || !hasCheck(checks, LevelWarning, "close to") {
		t.Fatalf("got %s %+v", verdict, checks)
	}

	// X allows larger GIFs than other images.
	if verdict, _ := evaluate(t, "x", Media{Filename: "anim.gif", SizeBytes: 10 << 20, Width: 500, Height: 500}); verdict != LevelOK {
		t.Fatalf("x gif verdict %s", verdict)
	}
}

func TestEvaluateFormatAndDimensions(t *testing.T) {
	if verdict, checks := evaluate(t, "bluesky", Media{Filename: "photo.heic", SizeBytes: 100}); verdict != LevelError || !hasCheck(checks, LevelError, ".heic") {
		t.Fatalf("got %s %+v", verdict, checks)
	}

	verdict, checks := evaluate(t, "x", Media{Filename: "huge.png", SizeBytes: 1000, Width: 9000, Height: 2000})
	if verdict != LevelError || !hasCheck(checks, LevelError, "larger than the 8192×8192") || !hasCheck(checks, LevelWarning, "wide") {
		t.Fatalf("got %s %+v", verdict, checks)
	}

	verdict, checks = evaluate(t, "mastodon", Media{Filename: "pano.jpg", SizeBytes: 1000, Width: 10000, Height: 4000})
	if verdict != LevelWarning || !hasCheck(checks, LevelWarning, "downscaled") {
		t.Fatalf("got %s %+v", verdict, checks)
	}

	verdict, checks = evaluate(t, "bluesky", Media{Filename: "big.jpg", SizeBytes: 1000, Width: 4000, Height: 3000})
	if verdict != LevelWarning || !hasCheck(checks, LevelWarning, "2000px") {
		t.Fatalf("got %s %+v", verdict, checks)
	}
}

func TestEvaluateVideoSkipsDimensions(t *testing.T) {
	verdict, checks := evaluate(t, "bluesky", Media{Filename: "clip.mp4", SizeBytes: 50_000_000})
	if verdict != LevelOK || len(checks) != 1 {
		t.Fatalf("got %s %+v", verdict, checks)
	}
	if verdict, _ := evaluate(t, "bluesky", Media{Filename: "clip.webm", SizeBytes: 1000}); verdict != LevelError {
		t.Fatalf("webm on bluesky: %s", verdict)
	}
}
