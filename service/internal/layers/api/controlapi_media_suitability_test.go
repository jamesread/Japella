package api

import (
	"testing"

	"github.com/jamesread/japella/internal/medialimits"
)

func TestPostingTargetIsStrictest(t *testing.T) {
	var networks []mediaNetwork
	for _, p := range []string{"mastodon", "bluesky", "x"} {
		l, _ := medialimits.ForProtocol(p)
		networks = append(networks, mediaNetwork{protocol: p, limits: l})
	}
	got := postingTarget(networks)
	if got.MaxBytes != 1_000_000 || got.MaxSide != 2000 {
		t.Fatalf("got %+v", got)
	}

	l, _ := medialimits.ForProtocol("mastodon")
	if got := postingTarget([]mediaNetwork{{protocol: "mastodon", limits: l}}); got.MaxBytes != 16<<20 || got.MaxSide != 5760 {
		t.Fatalf("mastodon only: %+v", got)
	}
}
