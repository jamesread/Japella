package medialimits

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvaluateGolden records Evaluate's full output for every platform across
// a matrix of media. Run with UPDATE_GOLDEN=1 to accept a deliberate change.
func TestEvaluateGolden(t *testing.T) {
	media := []Media{
		{Filename: "a.jpg", SizeBytes: 300_000, Width: 1600, Height: 900},
		{Filename: "b.png", SizeBytes: 0, Width: 0, Height: 0},
		{Filename: "c.png", SizeBytes: 950_000, Width: 1000, Height: 1000},
		{Filename: "d.png", SizeBytes: 30 << 20, Width: 9000, Height: 9000},
		{Filename: "e.jpg", SizeBytes: 1000, Width: 10, Height: 10},
		{Filename: "f.jpg", SizeBytes: 1000, Width: 10000, Height: 1000},
		{Filename: "g.jpg", SizeBytes: 1000, Width: 1000, Height: 5000},
		{Filename: "h.gif", SizeBytes: 4 << 20, Width: 500, Height: 500},
		{Filename: "i.gif", SizeBytes: 20 << 20, Width: 500, Height: 500},
		{Filename: "j.webp", SizeBytes: 1000, Width: 4000, Height: 3000},
		{Filename: "k.heic", SizeBytes: 1000, Width: 100, Height: 100},
		{Filename: "l.mp4", SizeBytes: 50_000_000},
		{Filename: "m.webm", SizeBytes: 1_000_000_000},
		{Filename: "n.MOV", SizeBytes: 1000},
		{Filename: "noext", SizeBytes: 1000, Width: 100, Height: 100},
	}

	protocols := make([]string, 0, len(byProtocol))
	for p := range byProtocol {
		protocols = append(protocols, p)
	}
	slices.Sort(protocols)

	var out strings.Builder
	for _, p := range protocols {
		for _, m := range media {
			verdict, checks := Evaluate(byProtocol[p], m)
			fmt.Fprintf(&out, "%s %s %d %dx%d => %s\n", p, m.Filename, m.SizeBytes, m.Width, m.Height, verdict)
			for _, c := range checks {
				fmt.Fprintf(&out, "  %s: %s\n", c.Level, c.Message)
			}
		}
	}

	golden := filepath.Join("testdata", "evaluate.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.MkdirAll("testdata", 0o750))
		require.NoError(t, os.WriteFile(golden, []byte(out.String()), 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.Equal(t, string(want), out.String())
}
