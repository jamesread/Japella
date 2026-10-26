package bluesky

import (
	"encoding/json"
	"testing"
)

func TestFlattenBlueskyThread(t *testing.T) {
	raw := `{"thread":{"$type":"app.bsky.feed.defs#threadViewPost",
		"post":{"uri":"at://did:plc:me/app.bsky.feed.post/root","author":{"handle":"me.bsky.social"},"record":{"text":"root"}},
		"replies":[
			{"$type":"app.bsky.feed.defs#threadViewPost",
			 "post":{"uri":"at://did:plc:a/app.bsky.feed.post/r1","author":{"handle":"alice.bsky.social","displayName":"Alice","avatar":"https://cdn/a.jpg"},"record":{"text":"first!","createdAt":"2026-10-01T12:00:00Z"}},
			 "replies":[
				{"$type":"app.bsky.feed.defs#threadViewPost",
				 "post":{"uri":"at://did:plc:me/app.bsky.feed.post/r1a","author":{"handle":"me.bsky.social"},"record":{"text":"thanks"}}}
			 ]},
			{"$type":"app.bsky.feed.defs#blockedPost","uri":"at://did:plc:b/app.bsky.feed.post/x","blocked":true}
		]}}`

	var res bskyGetPostThreadResponse
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatal(err)
	}

	root := res.Thread.Post.URI
	replies := flattenBlueskyThread(res.Thread, root, nil)
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies (blocked skipped), got %d", len(replies))
	}

	r1 := replies[0]
	if r1.ParentRemoteID != root || r1.AuthorName != "Alice" || r1.Content != "first!" || r1.PostedDate.IsZero() {
		t.Fatalf("unexpected first reply: %+v", r1)
	}
	if r1.RemoteURL != "https://bsky.app/profile/alice.bsky.social/post/r1" {
		t.Fatalf("unexpected web url %q", r1.RemoteURL)
	}
	if replies[1].ParentRemoteID != r1.RemoteID {
		t.Fatalf("nested reply should point at r1, got %q", replies[1].ParentRemoteID)
	}
}

func TestReplyRefFor(t *testing.T) {
	top := &bskyPostView{URI: "at://a/app.bsky.feed.post/1", CID: "c1"}
	ref := replyRefFor(top)
	if ref.Root != ref.Parent || ref.Parent.URI != top.URI || ref.Parent.CID != "c1" {
		t.Fatalf("replying to a top-level post: root should be the parent, got %+v", ref)
	}

	nested := &bskyPostView{URI: "at://b/app.bsky.feed.post/2", CID: "c2"}
	nested.Record.Reply = &BlueskyReplyRef{Root: BlueskyStrongRef{URI: top.URI, CID: "c1"}}
	ref = replyRefFor(nested)
	if ref.Root.URI != top.URI || ref.Parent.URI != nested.URI {
		t.Fatalf("replying to a reply: root should be the thread root, got %+v", ref)
	}
}
