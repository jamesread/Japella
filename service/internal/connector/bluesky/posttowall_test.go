package bluesky

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePDS answers the two XRPC calls posting makes.
type fakePDS struct {
	uploads    atomic.Int32
	failUpload bool
	created    []byte
}

func (f *fakePDS) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/xrpc/com.atproto.repo.uploadBlob":
			if f.failUpload {
				http.Error(w, `{"error":"BlobTooLarge"}`, http.StatusBadRequest)
				return
			}
			n := f.uploads.Add(1)
			_, _ = fmt.Fprintf(w, `{"blob":{"ref":{"$link":"bafy%d"},"mimeType":%q,"size":%d}}`, n, r.Header.Get("Content-Type"), len(body))
		case "/xrpc/com.atproto.repo.createRecord":
			f.created = body
			_, _ = w.Write([]byte(`{"uri":"at://did:plc:abc/app.bsky.feed.post/3kpost","cid":"bafycid"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func writePNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))))
	require.NoError(t, f.Close())
	return path
}

func testClient() *utils.ChainingHttpClient {
	return utils.NewClient(log.NewEntry(log.New()))
}

func TestBuildPostRecordPlainTextHasNoEmbed(t *testing.T) {
	pds := (&fakePDS{}).start(t)
	record, err := (&BlueskyConnector{}).buildPostRecord(pds, testClient(), "just words", nil)
	require.NoError(t, err)
	assert.Equal(t, "app.bsky.feed.post", record.Type)
	assert.Equal(t, "just words", record.Text)
	assert.Nil(t, record.Embed)

	raw, err := json.Marshal(record)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"embed"`)
}

func TestBuildPostRecordUploadsAtMostFourImages(t *testing.T) {
	fake := &fakePDS{}
	pds := fake.start(t)
	dir := t.TempDir()
	paths := []string{
		writePNG(t, dir, "a.png", 40, 20),
		writePNG(t, dir, "b.png", 10, 10),
		writePNG(t, dir, "c.png", 10, 10),
		writePNG(t, dir, "d.png", 10, 10),
		writePNG(t, dir, "e.png", 10, 10),
	}

	record, err := (&BlueskyConnector{}).buildPostRecord(pds, testClient(), "pics", paths)
	require.NoError(t, err)
	embed, ok := record.Embed.(*BlueskyEmbedImages)
	require.True(t, ok, "embed is %T", record.Embed)
	assert.Equal(t, "app.bsky.embed.images", embed.Type)
	require.Len(t, embed.Images, 4)
	assert.Equal(t, int32(4), fake.uploads.Load())
	assert.Equal(t, "bafy1", embed.Images[0].Image.Ref.Link)
	assert.Equal(t, &BlueskyAspectRatio{Width: 40, Height: 20}, embed.Images[0].AspectRatio)
}

func TestBuildPostRecordFailsWhenUploadFails(t *testing.T) {
	pds := (&fakePDS{failUpload: true}).start(t)
	path := writePNG(t, t.TempDir(), "a.png", 10, 10)
	_, err := (&BlueskyConnector{}).buildPostRecord(pds, testClient(), "pics", []string{path})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestBuildPostRecordLinkCard(t *testing.T) {
	pds := (&fakePDS{}).start(t)
	page := previewServer(t, http.StatusOK, "text/html", `<meta property="og:title" content="Card title"><meta property="og:description" content="Card text">`)

	record, err := (&BlueskyConnector{}).buildPostRecord(pds, testClient(), "read "+page+"/story now", nil)
	require.NoError(t, err)
	embed, ok := record.Embed.(*BlueskyEmbedExternal)
	require.True(t, ok, "embed is %T", record.Embed)
	assert.Equal(t, BlueskyExternal{URI: page + "/story", Title: "Card title", Description: "Card text"}, embed.External)
}

func TestBuildPostRecordLinkWithoutPreviewHasNoEmbed(t *testing.T) {
	pds := (&fakePDS{}).start(t)
	page := previewServer(t, http.StatusNotFound, "text/html", "")
	record, err := (&BlueskyConnector{}).buildPostRecord(pds, testClient(), "dead link "+page, nil)
	require.NoError(t, err)
	assert.Nil(t, record.Embed)
}

func TestPublishRecord(t *testing.T) {
	fake := &fakePDS{}
	pds := fake.start(t)
	req := &BlueskyCreatePostRequest{Repo: "did:plc:abc", Collection: "app.bsky.feed.post", Record: BlueskyPostRecord{Type: "app.bsky.feed.post", Text: "hello"}}

	res, err := (&BlueskyConnector{}).publishRecord(testClient(), pds+"/xrpc/com.atproto.repo.createRecord", req, nil)
	require.NoError(t, err)
	assert.Equal(t, "at://did:plc:abc/app.bsky.feed.post/3kpost", res.Uri)

	var sent BlueskyCreatePostRequest
	require.NoError(t, json.Unmarshal(fake.created, &sent))
	assert.Equal(t, "did:plc:abc", sent.Repo)
	assert.Equal(t, "app.bsky.feed.post", sent.Collection)
}

func TestPostWebURL(t *testing.T) {
	b := &BlueskyConnector{}
	cases := []struct {
		identity, did, uri, want string
	}{
		{"me.bsky.social", "did:plc:abc", "at://did:plc:abc/app.bsky.feed.post/3kpost", "https://bsky.app/profile/me.bsky.social/post/3kpost"},
		{"", "did:plc:abc", "at://did:plc:abc/app.bsky.feed.post/3kpost", "https://bsky.app/profile/did:plc:abc/post/3kpost"},
		{"me.bsky.social", "", "norkey", "https://bsky.app/profile/me.bsky.social/post/norkey"},
		{"me.bsky.social", "", "at://x/", "https://bsky.app/profile/me.bsky.social/post/at://x/"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, b.postWebURL(&connector.SocialAccount{Identity: c.identity, Did: c.did}, c.uri), c.uri)
	}
}
