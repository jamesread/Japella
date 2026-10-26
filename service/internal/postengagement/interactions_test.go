package postengagement

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jamesread/japella/internal/db"
)

func TestSummariseInteractions(t *testing.T) {
	today := time.Date(2026, 10, 7, 18, 0, 0, 0, time.UTC)
	day := func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }

	post := &db.Post{Content: "hello", PostURL: sql.NullString{String: "https://x/1", Valid: true}}
	post.ID = 1
	post.CreatedAt = day(5, 9)

	src := InteractionSources{
		Posts: []*db.Post{post},
		Events: []*db.PostEngagementEvent{
			{PostID: 1, Likes: 3, Replies: 2, DetectedAt: day(6, 10)},
			{PostID: 1, Replies: 1, DetectedAt: day(7, 10)}, // replies only: hidden when threads are used
		},
		Replies: []*db.AccountPostReply{
			{PostReply: db.PostReply{PostID: 1, RemoteID: "r1", PostedAt: sql.NullTime{Time: day(6, 11), Valid: true}}},
			{PostReply: db.PostReply{PostID: 1, RemoteID: "r2", PostedAt: sql.NullTime{Time: day(7, 12), Valid: true}}},
		},
		RepliesFromThreads: true,
	}

	sum := SummariseInteractions(today, 7, src, 0)

	if sum.Posts != 1 || sum.Counts.Likes != 3 || sum.Counts.Replies != 2 {
		t.Fatalf("unexpected totals: posts=%d counts=%+v", sum.Posts, sum.Counts)
	}

	kinds := []string{}
	for _, it := range sum.Items {
		kinds = append(kinds, it.Kind)
	}
	want := []string{InteractionReply, InteractionReply, InteractionEngagement, InteractionPost}
	if len(kinds) != len(want) {
		t.Fatalf("got items %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("got items %v, want %v", kinds, want)
		}
	}
	if sum.Items[2].Counts.Replies != 0 || sum.Items[2].Counts.Likes != 3 {
		t.Fatalf("engagement item should drop replies when threads are used: %+v", sum.Items[2].Counts)
	}

	if len(sum.Daily) != 7 || sum.Daily[4].Posts != 1 || sum.Daily[5].Counts.Interactions() != 4 || sum.Daily[6].Counts.Interactions() != 1 {
		t.Fatalf("unexpected daily buckets: %+v", sum.Daily)
	}
}

func TestSummariseInteractionsWithoutThreads(t *testing.T) {
	today := time.Date(2026, 10, 7, 18, 0, 0, 0, time.UTC)
	src := InteractionSources{
		Events: []*db.PostEngagementEvent{{PostID: 1, Replies: 2, DetectedAt: today}},
	}

	sum := SummariseInteractions(today, 7, src, 1)
	if sum.Counts.Replies != 2 || len(sum.Items) != 1 || sum.Items[0].Counts.Replies != 2 {
		t.Fatalf("reply counts should come from events: %+v", sum)
	}
}

func TestSummariseInteractionsActivity(t *testing.T) {
	today := time.Date(2026, 10, 7, 18, 0, 0, 0, time.UTC)
	day := func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }

	src := InteractionSources{
		Activity: []*db.AccountActivity{
			{Kind: db.AccountActivityLiked, PostSnippet: "nice", PostAuthor: "Alice", CreatedAt: day(7, 9)},
			{Kind: db.AccountActivityCommented, CommentContent: sql.NullString{String: "agreed", Valid: true}, CreatedAt: day(7, 10)},
			{Kind: db.AccountActivityLiked, CreatedAt: day(6, 9)},
			{Kind: "unknown", CreatedAt: day(7, 11)},
		},
	}

	sum := SummariseInteractions(today, 7, src, 0)

	if sum.LikesGiven != 2 || sum.CommentsMade != 1 {
		t.Fatalf("given = %d likes, %d comments; want 2, 1", sum.LikesGiven, sum.CommentsMade)
	}
	if sum.Counts.Interactions() != 0 {
		t.Fatalf("outgoing activity counted as received: %+v", sum.Counts)
	}
	if len(sum.Items) != 3 || sum.Items[0].Kind != InteractionCommented || sum.Items[0].Comment != "agreed" {
		t.Fatalf("unexpected items: %+v", sum.Items)
	}
	last := sum.Daily[len(sum.Daily)-1]
	if last.LikesGiven != 1 || last.CommentsMade != 1 {
		t.Fatalf("today bucket = %+v", last)
	}
}
