package bluesky

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeAppView resolves alice.bsky.social and nothing else.
func fakeAppView(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("handle") == "alice.bsky.social" {
			_, _ = w.Write([]byte(`{"did":"did:plc:alice"}`))
			return
		}
		http.Error(w, `{"error":"InvalidRequest"}`, http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	orig := appViewURL
	appViewURL = srv.URL
	t.Cleanup(func() { appViewURL = orig })
}

func facet(start, end int, f BlueskyFacetFeature) BlueskyFacet {
	return BlueskyFacet{Index: BlueskyFacetIndex{ByteStart: start, ByteEnd: end}, Features: []BlueskyFacetFeature{f}}
}

func TestParseFacets(t *testing.T) {
	fakeAppView(t)
	b := &BlueskyConnector{}

	link := func(s, e int, uri string) BlueskyFacet {
		return facet(s, e, BlueskyFacetFeature{Type: "app.bsky.richtext.facet#link", URI: uri})
	}
	mention := func(s, e int, did string) BlueskyFacet {
		return facet(s, e, BlueskyFacetFeature{Type: "app.bsky.richtext.facet#mention", DID: did})
	}
	tag := func(s, e int, name string) BlueskyFacet {
		return facet(s, e, BlueskyFacetFeature{Type: "app.bsky.richtext.facet#tag", Tag: name})
	}

	cases := []struct {
		name string
		text string
		want []BlueskyFacet
	}{
		{"empty", "", nil},
		{"plain text", "nothing to see", []BlueskyFacet{}},
		{"link", "go https://example.com/a?b=1 now", []BlueskyFacet{link(3, 28, "https://example.com/a?b=1")}},
		{"mention resolved", "hi @alice.bsky.social!", []BlueskyFacet{mention(3, 21, "did:plc:alice")}},
		{"mention unresolved keeps facet without DID", "hi @bob.example.com", []BlueskyFacet{mention(3, 19, "")}},
		{"hashtag", "#golang rocks", []BlueskyFacet{tag(0, 7, "golang")}},
		{"unicode hashtag uses byte offsets", "café #été ok", []BlueskyFacet{tag(6, 12, "été")}},
		{
			"links first, then mentions, then tags",
			"#a @alice.bsky.social https://x.example/#frag",
			[]BlueskyFacet{link(22, 45, "https://x.example/#frag"), mention(3, 21, "did:plc:alice"), tag(0, 2, "a")},
		},
		{"no tag or mention inside a link", "https://x.example/@alice.bsky.social/#tag", []BlueskyFacet{link(0, 41, "https://x.example/@alice.bsky.social/#tag")}},
		{"invalid URL is not a link", "see http://%zz", []BlueskyFacet{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, b.parseFacets(c.text))
		})
	}
}
