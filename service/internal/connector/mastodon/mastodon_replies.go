package mastodon

import (
	"fmt"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

type statusContext struct {
	Descendants []contextStatus `json:"descendants"`
}

type contextStatus struct {
	ID          string    `json:"id"`
	InReplyToID string    `json:"in_reply_to_id"`
	URL         string    `json:"url"`
	URI         string    `json:"uri"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	Account     struct {
		Acct        string `json:"acct"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Avatar      string `json:"avatar"`
	} `json:"account"`
}

func (s contextStatus) toPostReply() *connector.PostReply {
	url := s.URL
	if url == "" {
		url = s.URI
	}
	return &connector.PostReply{
		RemoteID:        s.ID,
		ParentRemoteID:  s.InReplyToID,
		RemoteURL:       url,
		AuthorName:      s.Account.DisplayName,
		AuthorHandle:    "@" + s.Account.Acct,
		AuthorAvatarURL: s.Account.Avatar,
		Content:         utils.HTMLToText(s.Content),
		PostedDate:      s.CreatedAt,
	}
}

// FetchPostReplies loads the replies under a status this account published.
func (c *MastodonConnector) FetchPostReplies(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostReplies, error) {
	statusID, err := extractMastodonStatusID(remoteID, postURL)
	if err != nil {
		return nil, err
	}

	instanceURL := socialAccount.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	client := utils.NewClient(c.Logger()).Get(instanceURL + "/api/v1/statuses/" + statusID + "/context")
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(socialAccount.OAuthToken)

	var ctx statusContext
	client.AsJson(&ctx)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch mastodon replies for %s: %w", statusID, client.Err)
	}

	replies := make([]*connector.PostReply, 0, len(ctx.Descendants))
	for _, d := range ctx.Descendants {
		if d.ID != "" {
			replies = append(replies, d.toPostReply())
		}
	}

	return &connector.PostReplies{RootRemoteID: statusID, Replies: replies}, nil
}
