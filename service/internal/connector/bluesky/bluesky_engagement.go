package bluesky

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

// Public AppView: post views (including counts) don't require auth.
const bskyPublicAppView = "https://public.api.bsky.app"

type bskyGetPostsResponse struct {
	Posts []bskyPostView `json:"posts"`
}

type bskyPostView struct {
	URI         string `json:"uri"`
	LikeCount   uint32 `json:"likeCount"`
	RepostCount uint32 `json:"repostCount"`
	ReplyCount  uint32 `json:"replyCount"`
	QuoteCount  uint32 `json:"quoteCount"`
}

func (p bskyPostView) toPostEngagement() *connector.PostEngagement {
	return &connector.PostEngagement{
		Likes:   p.LikeCount,
		Reposts: p.RepostCount,
		Replies: p.ReplyCount,
		Quotes:  p.QuoteCount,
	}
}

// parseBlueskyPostRef extracts the repo (handle or DID) and record key from
// either an at:// URI or a https://bsky.app/profile/<repo>/post/<rkey> URL.
func parseBlueskyPostRef(ref string) (repo string, rkey string, err error) {
	ref = strings.TrimSpace(ref)

	if strings.HasPrefix(ref, "at://") {
		parts := strings.Split(strings.TrimPrefix(ref, "at://"), "/")
		if len(parts) == 3 && parts[1] == "app.bsky.feed.post" && parts[0] != "" && parts[2] != "" {
			return parts[0], parts[2], nil
		}
		return "", "", fmt.Errorf("unsupported bluesky at-uri %q", ref)
	}

	u, parseErr := url.Parse(ref)
	if parseErr != nil {
		return "", "", fmt.Errorf("invalid bluesky post url %q: %w", ref, parseErr)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 4 && parts[0] == "profile" && parts[2] == "post" && parts[1] != "" && parts[3] != "" {
		return parts[1], parts[3], nil
	}

	return "", "", fmt.Errorf("cannot determine bluesky post from %q", ref)
}

// FetchPostEngagement loads like/repost/reply/quote counts for a post this account published.
func (b *BlueskyConnector) FetchPostEngagement(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostEngagement, error) {
	ref := remoteID
	if ref == "" {
		ref = postURL
	}

	repo, rkey, err := parseBlueskyPostRef(ref)
	if err != nil {
		return nil, err
	}

	did := repo
	if !strings.HasPrefix(did, "did:") {
		// Post URLs are built from the account's handle; prefer the stored DID
		// so a later handle change doesn't break lookups.
		if socialAccount != nil && strings.HasPrefix(socialAccount.Did, "did:") {
			did = socialAccount.Did
		} else {
			did, err = b.resolveHandleToDID(repo)
			if err != nil {
				return nil, err
			}
		}
	}

	atURI := "at://" + did + "/app.bsky.feed.post/" + rkey

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

	if len(response.Posts) == 0 {
		return nil, fmt.Errorf("bluesky post %s not found", atURI)
	}

	return response.Posts[0].toPostEngagement(), nil
}
