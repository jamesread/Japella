package postreplies

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func row(id, parent string, minutes int) *db.PostReply {
	return &db.PostReply{
		RemoteID:       id,
		ParentRemoteID: parent,
		PostedAt:       sql.NullTime{Time: t0.Add(time.Duration(minutes) * time.Minute), Valid: true},
	}
}

func ids(nodes []*Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Reply.RemoteID
	}
	return out
}

func assertIDs(t *testing.T, label string, got []*Node, want ...string) {
	t.Helper()
	g := ids(got)
	if len(g) != len(want) {
		t.Fatalf("%s: got %v, want %v", label, g, want)
	}
	for i := range want {
		if g[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", label, g, want)
		}
	}
}

func TestShouldFetch(t *testing.T) {
	if ShouldFetch(nil, 0) {
		t.Fatal("no replies yet: nothing to fetch")
	}
	if !ShouldFetch(nil, 2) {
		t.Fatal("replies but never fetched: fetch")
	}

	fetched := &db.PostEngagement{RepliesFetchedAt: sql.NullTime{Time: t0, Valid: true}, RepliesFetchedCount: 3}
	if ShouldFetch(fetched, 3) {
		t.Fatal("counter unchanged: skip")
	}
	if !ShouldFetch(fetched, 4) || !ShouldFetch(fetched, 2) {
		t.Fatal("counter moved: refetch")
	}
}

func TestToDBNormalisesRootAndDropsDuplicates(t *testing.T) {
	rows := ToDB(7, &connector.PostReplies{
		RootRemoteID: "root",
		Replies: []*connector.PostReply{
			{RemoteID: "a", ParentRemoteID: "root", Content: "  hi  ", PostedDate: t0},
			{RemoteID: "b", ParentRemoteID: "a"},
			{RemoteID: "a", ParentRemoteID: "root"},
			{RemoteID: ""},
			{RemoteID: "root"},
		},
	})

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].PostID != 7 || rows[0].ParentRemoteID != "" || rows[0].Content != "hi" || !rows[0].PostedAt.Valid {
		t.Fatalf("unexpected top-level row: %+v", rows[0])
	}
	if rows[1].ParentRemoteID != "a" || rows[1].PostedAt.Valid {
		t.Fatalf("unexpected nested row: %+v", rows[1])
	}
}

func TestBuildThreadNestsAndOrders(t *testing.T) {
	roots := BuildThread([]*db.PostReply{
		row("late", "", 30),
		row("a2", "a", 20),
		row("a", "", 5),
		row("a1", "a", 10),
		row("a1x", "a1", 15),
	})

	assertIDs(t, "roots", roots, "a", "late")
	assertIDs(t, "a's replies", roots[0].Replies, "a1", "a2")
	assertIDs(t, "a1's replies", roots[0].Replies[0].Replies, "a1x")
}

func TestBuildThreadOrphansAndCyclesAreTopLevel(t *testing.T) {
	roots := BuildThread([]*db.PostReply{
		row("orphan", "deleted", 1),
		row("x", "y", 2),
		row("y", "x", 3),
		row("self", "self", 4),
	})

	assertIDs(t, "roots", roots, "orphan", "x", "y", "self")
	for _, n := range roots {
		if len(n.Replies) != 0 {
			t.Fatalf("%s should have no children, got %v", n.Reply.RemoteID, ids(n.Replies))
		}
	}
}
