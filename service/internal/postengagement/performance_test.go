package postengagement

import (
	"testing"
	"time"
)

func TestDailyBucketsWindowOldestFirst(t *testing.T) {
	today := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	samples := []PostSample{
		{PostID: 1, PostedAt: time.Date(2026, 10, 26, 8, 0, 0, 0, time.UTC), Counts: Counts{Likes: 2}},
		{PostID: 2, PostedAt: time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC), Counts: Counts{Replies: 1}},
		{PostID: 3, PostedAt: time.Date(2026, 10, 20, 23, 59, 0, 0, time.UTC), Counts: Counts{Reposts: 4}},
		{PostID: 4, PostedAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC), Counts: Counts{Likes: 100}}, // outside window
	}

	got := Daily(today, 7, samples)
	if len(got) != 7 {
		t.Fatalf("len = %d, want 7", len(got))
	}
	if got[0].Date != "2026-10-20" || got[6].Date != "2026-10-26" {
		t.Fatalf("window = %s..%s", got[0].Date, got[6].Date)
	}
	if got[0].Posts != 1 || got[0].Counts.Reposts != 4 {
		t.Fatalf("first day = %+v", got[0])
	}
	if got[6].Posts != 2 || got[6].Counts.Likes != 2 || got[6].Counts.Replies != 1 {
		t.Fatalf("today = %+v", got[6])
	}
	for _, b := range got[1:6] {
		if b.Posts != 0 || b.Counts != (Counts{}) {
			t.Fatalf("expected empty bucket, got %+v", b)
		}
	}
}

func TestDailyZeroDays(t *testing.T) {
	if got := Daily(time.Now(), 0, nil); len(got) != 0 {
		t.Fatalf("got %d buckets", len(got))
	}
}

func TestTotalAndInteractions(t *testing.T) {
	total := Total([]PostSample{
		{Counts: Counts{Likes: 1, Reposts: 2, Impressions: 50}},
		{Counts: Counts{Replies: 3, Quotes: 4, Impressions: 25}},
	})
	want := Counts{Likes: 1, Reposts: 2, Replies: 3, Quotes: 4, Impressions: 75}
	if total != want {
		t.Fatalf("total = %+v, want %+v", total, want)
	}
	if total.Interactions() != 10 {
		t.Fatalf("interactions = %d", total.Interactions())
	}
}

func TestSortByInteractions(t *testing.T) {
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	samples := []PostSample{
		{PostID: 1, PostedAt: base, Counts: Counts{Likes: 1}},
		{PostID: 2, PostedAt: base, Counts: Counts{Likes: 5}},
		{PostID: 3, PostedAt: base, Counts: Counts{Likes: 1, Impressions: 10}},
		{PostID: 4, PostedAt: base.Add(time.Hour), Counts: Counts{Likes: 1}},
	}
	SortByInteractions(samples)
	order := []uint32{samples[0].PostID, samples[1].PostID, samples[2].PostID, samples[3].PostID}
	want := []uint32{2, 3, 4, 1}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
