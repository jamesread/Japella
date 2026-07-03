package mastodon

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimelineStatusToFeedPostOriginal(t *testing.T) {
	status := TimelineStatus{
		ID:        "116840544724491003",
		URI:       "https://tooter.wishy.co.uk/users/wishy/statuses/116840544724491003",
		Content:   "<p>Hello world</p>",
		CreatedAt: time.Date(2026, 6, 30, 18, 47, 5, 0, time.UTC),
		Account: TimelineAccount{
			ID:       "109326084383865077",
			Username: "wishy",
			Avatar:   "https://tooter.wishy.co.uk/avatar.jpg",
			URI:      "https://tooter.wishy.co.uk/users/wishy",
		},
	}

	post, ok := timelineStatusToFeedPost(status)
	if !ok {
		t.Fatal("expected post to be accepted")
	}

	if post.AuthorID != "109326084383865077" {
		t.Fatalf("author id = %q, want full Mastodon snowflake", post.AuthorID)
	}
	if post.AuthorName != "wishy" {
		t.Fatalf("author name = %q", post.AuthorName)
	}
	if post.Content != "<p>Hello world</p>" {
		t.Fatalf("content = %q", post.Content)
	}
	if post.RemoteID != "116840544724491003" {
		t.Fatalf("remote id = %q", post.RemoteID)
	}
}

func TestTimelineStatusToFeedPostReblog(t *testing.T) {
	status := TimelineStatus{
		ID:        "wrapper-id",
		URI:       "https://example.social/users/wishy/statuses/wrapper-id/activity",
		Content:   "",
		CreatedAt: time.Date(2026, 6, 30, 19, 0, 0, 0, time.UTC),
		Account: TimelineAccount{
			ID:       "109326084383865077",
			Username: "wishy",
			URI:      "https://tooter.wishy.co.uk/users/wishy",
		},
		Reblog: &TimelineStatus{
			ID:      "original-id",
			URI:     "https://other.social/users/neil/statuses/original-id",
			Content: "<p>Original post</p>",
			Account: TimelineAccount{
				ID:       "999",
				Username: "neil",
			},
		},
	}

	post, ok := timelineStatusToFeedPost(status)
	if !ok {
		t.Fatal("expected reblog to be accepted")
	}

	if post.AuthorID != "109326084383865077" {
		t.Fatalf("author id = %q, want booster", post.AuthorID)
	}
	if post.AuthorName != "wishy" {
		t.Fatalf("author name = %q", post.AuthorName)
	}
	if post.Content != "<p>Original post</p>" {
		t.Fatalf("content = %q", post.Content)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(post.RemoteID), &parsed); err != nil {
		t.Fatalf("remote id should be announce json: %v", err)
	}
	if parsed["type"] != "Announce" {
		t.Fatalf("remote id type = %q", parsed["type"])
	}
}

func TestExtractMastodonStatusID(t *testing.T) {
	id, err := extractMastodonStatusID("116840544724491003", "")
	if err != nil || id != "116840544724491003" {
		t.Fatalf("numeric remote id: got %q err %v", id, err)
	}

	id, err = extractMastodonStatusID(`{"type":"Announce"}`, "https://tooter.wishy.co.uk/users/wishy/statuses/116840544724491003/activity")
	if err != nil || id != "116840544724491003" {
		t.Fatalf("announce remote id: got %q err %v", id, err)
	}

	_, err = extractMastodonStatusID(`{"type":"Announce"}`, "")
	if err == nil {
		t.Fatal("expected error when status id cannot be determined")
	}
}
