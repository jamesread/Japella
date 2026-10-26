package bluesky

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
)

// fetchPostView loads one post (with its cid and reply refs) from the public AppView.
func (b *BlueskyConnector) fetchPostView(atURI string) (*bskyPostView, error) {
	q := url.Values{}
	q.Set("uris", atURI)

	client := utils.NewClient(b.Logger()).Get(bskyPublicAppView + "/xrpc/app.bsky.feed.getPosts?" + q.Encode())
	if client.Err != nil {
		return nil, client.Err
	}

	var response bskyGetPostsResponse
	client.AsJson(&response)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch bluesky post %s: %w", atURI, client.Err)
	}
	if len(response.Posts) == 0 || response.Posts[0].CID == "" {
		return nil, fmt.Errorf("bluesky post %s not found", atURI)
	}
	return &response.Posts[0], nil
}

// replyRefFor builds the reply refs for answering parent: the root is the
// parent's own root when it is itself a reply.
func replyRefFor(parent *bskyPostView) *BlueskyReplyRef {
	parentRef := BlueskyStrongRef{URI: parent.URI, CID: parent.CID}
	root := parentRef
	if parent.Record.Reply != nil && parent.Record.Reply.Root.URI != "" && parent.Record.Reply.Root.CID != "" {
		root = parent.Record.Reply.Root
	}
	return &BlueskyReplyRef{Root: root, Parent: parentRef}
}

// createRecord writes a record to the account's PDS through its DPoP
// client, refreshing the token and retrying once if it has expired. Same
// approach as PostToWall, without its media handling.
func (b *BlueskyConnector) createRecord(dbSocialAccount *db.SocialAccount, collection string, record any) (string, error) {
	pdsURL, err := b.getPDSEndpoint(dbSocialAccount)
	if err != nil {
		return "", fmt.Errorf("failed to get PDS endpoint: %w", err)
	}

	client := b.ensureDPoPClient(dbSocialAccount)
	if client == nil {
		return "", fmt.Errorf("could not create an authenticated Bluesky client; try reconnecting the account")
	}

	req := map[string]any{"repo": dbSocialAccount.Did, "collection": collection, "record": record}
	res := BlueskyCreatePostResponse{}
	endpoint := pdsURL + "/xrpc/com.atproto.repo.createRecord"

	client.PostWithJson(endpoint, req)
	client.AsJson(&res)

	if client.Err != nil && strings.Contains(client.Err.Error()+string(client.ResBody), "expired") {
		if refreshErr := b.refreshToken(dbSocialAccount); refreshErr != nil {
			return "", fmt.Errorf("token refresh failed: %w", refreshErr)
		}
		updated, getErr := b.db.GetSocialAccount(dbSocialAccount.ID)
		if getErr != nil || updated == nil {
			return "", fmt.Errorf("failed to reload social account after token refresh")
		}
		if client = b.ensureDPoPClient(updated); client == nil {
			return "", fmt.Errorf("could not create an authenticated Bluesky client after token refresh")
		}
		client.PostWithJson(endpoint, req)
		client.AsJson(&res)
	}

	if client.Err != nil {
		if len(client.ResBody) > 0 {
			return "", fmt.Errorf("%w: %s", client.Err, string(client.ResBody))
		}
		return "", client.Err
	}
	return res.Uri, nil
}

// ReplyToPost publishes a reply to any Bluesky post as the account.
func (b *BlueskyConnector) ReplyToPost(socialAccount *connector.SocialAccount, remoteID string, remoteURL string, content string) (*connector.PostReply, error) {
	parentURI, err := b.publishedPostATURI(socialAccount, remoteID, remoteURL)
	if err != nil {
		return nil, err
	}
	parent, err := b.fetchPostView(parentURI)
	if err != nil {
		return nil, err
	}

	dbSocialAccount, err := b.db.GetSocialAccount(socialAccount.Id)
	if err != nil || dbSocialAccount == nil {
		return nil, fmt.Errorf("failed to load social account %d", socialAccount.Id)
	}

	now := time.Now().UTC()
	record := BlueskyPostRecord{
		Type:      "app.bsky.feed.post",
		Text:      content,
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
		Facets:    b.parseFacets(content),
		Reply:     replyRefFor(parent),
	}

	uri, err := b.createRecord(dbSocialAccount, "app.bsky.feed.post", record)
	if err != nil {
		return nil, fmt.Errorf("failed to post bluesky reply: %w", err)
	}

	return &connector.PostReply{
		RemoteID:       uri,
		ParentRemoteID: parent.URI,
		RemoteURL:      blueskyWebURL(uri, socialAccount.Identity),
		AuthorHandle:   socialAccount.Identity,
		Content:        content,
		PostedDate:     now,
	}, nil
}
