package bluesky

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
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

	CID    string `json:"cid"`
	Record struct {
		Reply *BlueskyReplyRef `json:"reply"`
	} `json:"record"`
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
	if after, ok := strings.CutPrefix(ref, "at://"); ok {
		return parseBlueskyATURI(after, ref)
	}
	return parseBlueskyWebURL(ref)
}

func parseBlueskyATURI(after, original string) (repo string, rkey string, err error) {
	parts := strings.Split(after, "/")
	if len(parts) == 3 && parts[1] == "app.bsky.feed.post" && parts[0] != "" && parts[2] != "" {
		return parts[0], parts[2], nil
	}
	return "", "", fmt.Errorf("unsupported bluesky at-uri %q", original)
}

func parseBlueskyWebURL(ref string) (repo string, rkey string, err error) {
	u, parseErr := url.Parse(ref)
	if parseErr != nil {
		return "", "", fmt.Errorf("invalid bluesky post url %q: %w", ref, parseErr)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if isBlueskyProfilePostPath(parts) {
		return parts[1], parts[3], nil
	}
	return "", "", fmt.Errorf("cannot determine bluesky post from %q", ref)
}

func isBlueskyProfilePostPath(parts []string) bool {
	return len(parts) == 4 && parts[0] == "profile" && parts[2] == "post" && parts[1] != "" && parts[3] != ""
}

// publishedPostATURI resolves the at:// URI of a post from either its remote
// id or its URL. It was written for posts this account published, but also
// serves other people's posts (feed comment threads).
func (b *BlueskyConnector) publishedPostATURI(socialAccount *connector.SocialAccount, remoteID string, postURL string) (string, error) {
	ref := remoteID
	if ref == "" {
		ref = postURL
	}
	repo, rkey, err := parseBlueskyPostRef(ref)
	if err != nil {
		return "", err
	}
	did, err := b.resolvePostRepoDID(socialAccount, repo)
	if err != nil {
		return "", err
	}
	return "at://" + did + "/app.bsky.feed.post/" + rkey, nil
}

func (b *BlueskyConnector) resolvePostRepoDID(socialAccount *connector.SocialAccount, repo string) (string, error) {
	if strings.HasPrefix(repo, "did:") {
		return repo, nil
	}
	if did := didFromMatchingAccount(socialAccount, repo); did != "" {
		return did, nil
	}
	return b.resolveHandleToDID(repo)
}

func didFromMatchingAccount(socialAccount *connector.SocialAccount, repo string) string {
	if socialAccount == nil || !strings.HasPrefix(socialAccount.Did, "did:") {
		return ""
	}
	if !strings.EqualFold(strings.TrimPrefix(repo, "@"), strings.TrimPrefix(socialAccount.Identity, "@")) {
		return ""
	}
	return socialAccount.Did
}

// FetchPostEngagement loads like/repost/reply/quote counts for a post this account published.
func (b *BlueskyConnector) FetchPostEngagement(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostEngagement, error) {
	atURI, err := b.publishedPostATURI(socialAccount, remoteID, postURL)
	if err != nil {
		return nil, err
	}
	response, err := getPostsFromAppView(atURI, b.Logger())
	if err != nil {
		return nil, err
	}
	if len(response.Posts) == 0 {
		return nil, fmt.Errorf("bluesky post %s not found", atURI)
	}
	return response.Posts[0].toPostEngagement(), nil
}

func getPostsFromAppView(atURI string, logger *log.Entry) (*bskyGetPostsResponse, error) {
	q := url.Values{}
	q.Set("uris", atURI)
	client := utils.NewClient(logger).Get(bskyPublicAppView + "/xrpc/app.bsky.feed.getPosts?" + q.Encode())
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
	return &response, nil
}
