package bluesky

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

// bskyThreadDepth is how many levels of replies are requested; the AppView
// caps this itself, deeper replies are simply not returned.
const bskyThreadDepth = 10

type bskyGetPostThreadResponse struct {
	Thread bskyThreadNode `json:"thread"`
}

// bskyThreadNode is a threadViewPost; blocked and not-found nodes have no Post.
type bskyThreadNode struct {
	Type    string           `json:"$type"`
	Post    *bskyThreadPost  `json:"post"`
	Replies []bskyThreadNode `json:"replies"`
}

type bskyThreadPost struct {
	URI    string `json:"uri"`
	Author struct {
		Did         string `json:"did"`
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
		Avatar      string `json:"avatar"`
	} `json:"author"`
	Record struct {
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"createdAt"`
	} `json:"record"`
}

// flattenBlueskyThread walks the replies under node, depth first.
func flattenBlueskyThread(node bskyThreadNode, parentURI string, out []*connector.PostReply) []*connector.PostReply {
	for _, child := range node.Replies {
		if child.Post == nil || child.Post.URI == "" {
			continue
		}
		p := child.Post
		out = append(out, &connector.PostReply{
			RemoteID:        p.URI,
			ParentRemoteID:  parentURI,
			RemoteURL:       blueskyWebURL(p.URI, p.Author.Handle),
			AuthorName:      p.Author.DisplayName,
			AuthorHandle:    p.Author.Handle,
			AuthorAvatarURL: p.Author.Avatar,
			Content:         p.Record.Text,
			PostedDate:      p.Record.CreatedAt,
		})
		out = flattenBlueskyThread(child, p.URI, out)
	}
	return out
}

// FetchPostReplies loads the reply thread under a post this account published.
func (b *BlueskyConnector) FetchPostReplies(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostReplies, error) {
	atURI, err := b.publishedPostATURI(socialAccount, remoteID, postURL)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("uri", atURI)
	q.Set("depth", strconv.Itoa(bskyThreadDepth))
	q.Set("parentHeight", "0")

	client := utils.NewClient(b.Logger()).Get(bskyPublicAppView + "/xrpc/app.bsky.feed.getPostThread?" + q.Encode())
	if client.Err != nil {
		return nil, client.Err
	}

	var response bskyGetPostThreadResponse
	client.AsJson(&response)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch bluesky thread %s: %w", atURI, client.Err)
	}

	if response.Thread.Post == nil {
		return nil, fmt.Errorf("bluesky post %s not found", atURI)
	}

	return &connector.PostReplies{
		RootRemoteID: response.Thread.Post.URI,
		Replies:      flattenBlueskyThread(response.Thread, response.Thread.Post.URI, nil),
	}, nil
}
