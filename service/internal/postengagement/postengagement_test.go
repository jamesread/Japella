package postengagement

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

func storedRow(likes, reposts, replies, quotes uint32) *db.PostEngagement {
	return &db.PostEngagement{
		PostID:        1,
		Likes:         likes,
		Reposts:       reposts,
		Replies:       replies,
		Quotes:        quotes,
		LastChangedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}
}

func TestCompareFirstFetchIsBaseline(t *testing.T) {
	cmp := Compare(nil, Counts{Likes: 10, Replies: 2})
	if !cmp.Baseline || !cmp.Changed {
		t.Fatalf("expected baseline+changed, got %+v", cmp)
	}
	if cmp.HasNewActivity() {
		t.Fatal("baseline fetch must not notify")
	}
}

func TestCompareErrorOnlyRowIsBaseline(t *testing.T) {
	errRow := &db.PostEngagement{PostID: 1, LastError: "boom"}
	cmp := Compare(errRow, Counts{Likes: 5})
	if !cmp.Baseline || cmp.HasNewActivity() {
		t.Fatalf("row with no successful fetch should be baseline, got %+v", cmp)
	}
}

func TestCompareDetectsNewActivity(t *testing.T) {
	cmp := Compare(storedRow(3, 1, 0, 0), Counts{Likes: 5, Reposts: 1, Replies: 2})
	if cmp.Baseline || !cmp.Changed || !cmp.HasNewActivity() {
		t.Fatalf("expected new activity, got %+v", cmp)
	}
	want := Delta{Likes: 2, Replies: 2}
	if cmp.Delta != want {
		t.Fatalf("delta = %+v, want %+v", cmp.Delta, want)
	}
}

func TestCompareDecreaseIsChangeButNotActivity(t *testing.T) {
	cmp := Compare(storedRow(5, 0, 0, 0), Counts{Likes: 4})
	if !cmp.Changed {
		t.Fatal("expected change to be recorded")
	}
	if cmp.HasNewActivity() || cmp.Delta.Likes != 0 {
		t.Fatalf("decrease must not notify, got %+v", cmp)
	}
}

func TestCompareImpressionsOnlyIsNotActivity(t *testing.T) {
	row := storedRow(1, 0, 0, 0)
	row.Impressions = 100
	cmp := Compare(row, Counts{Likes: 1, Impressions: 250})
	if !cmp.Changed || cmp.HasNewActivity() {
		t.Fatalf("impressions-only change should not notify, got %+v", cmp)
	}
	if cmp.Delta.Impressions != 150 {
		t.Fatalf("impressions delta = %d", cmp.Delta.Impressions)
	}
}

func TestCompareUnchanged(t *testing.T) {
	cmp := Compare(storedRow(2, 1, 1, 0), Counts{Likes: 2, Reposts: 1, Replies: 1})
	if cmp.Changed || cmp.HasNewActivity() {
		t.Fatalf("expected no change, got %+v", cmp)
	}
}

func TestFromConnectorNil(t *testing.T) {
	if FromConnector(nil) != (Counts{}) {
		t.Fatal("nil engagement should be zero counts")
	}
	got := FromConnector(&connector.PostEngagement{Likes: 1, Quotes: 2})
	if got.Likes != 1 || got.Quotes != 2 {
		t.Fatalf("got %+v", got)
	}
}
