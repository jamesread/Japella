package api

import (
	"strings"
	"testing"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
)

func TestClientTagSnapshotIsSanitised(t *testing.T) {
	ref := &controlv1.PostRef{RemoteUrl: "javascript:alert(1)", RemoteId: "123"}
	snap := clientTagSnapshot(&controlv1.FeedPost{
		Content:          `<p>hi <script>alert(1)</script></p><img src=x onerror=alert(1)>`,
		AuthorName:       "Mallory",
		AuthorAvatarUrl:  "data:image/png;base64,AAAA",
		AuthorProfileUrl: "https://example.social/@mallory",
		Id:               42,
		Tags:             []string{"forged"},
		CanLike:          true,
	}, ref)

	if strings.Contains(snap.Content, "<script") || strings.Contains(snap.Content, "<img") {
		t.Fatalf("content not escaped: %q", snap.Content)
	}
	if snap.RemoteUrl != "" || snap.AuthorAvatarUrl != "" {
		t.Fatalf("unsafe URLs kept: %q %q", snap.RemoteUrl, snap.AuthorAvatarUrl)
	}
	if snap.AuthorProfileUrl != "https://example.social/@mallory" || snap.RemoteId != "123" {
		t.Fatalf("safe fields lost: %+v", snap)
	}
	if snap.Id != 0 || len(snap.Tags) != 0 || snap.CanLike {
		t.Fatalf("client-controlled state kept: %+v", snap)
	}
}
