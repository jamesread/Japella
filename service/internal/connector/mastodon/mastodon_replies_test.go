package mastodon

import (
	"encoding/json"
	"testing"
)

func TestContextStatusToPostReply(t *testing.T) {
	raw := `{"descendants":[{"id":"2","in_reply_to_id":"1","url":"https://m/@a/2","content":"<p>hi</p>",
		"created_at":"2026-10-01T12:00:00.000Z","account":{"acct":"alice@other.social","display_name":"Alice","avatar":"https://m/a.png"}}]}`

	var ctx statusContext
	if err := json.Unmarshal([]byte(raw), &ctx); err != nil {
		t.Fatal(err)
	}

	r := ctx.Descendants[0].toPostReply()
	if r.RemoteID != "2" || r.ParentRemoteID != "1" || r.AuthorHandle != "@alice@other.social" || r.Content != "hi" || r.PostedDate.IsZero() {
		t.Fatalf("unexpected reply: %+v", r)
	}
}
