package mastodon

import (
	"fmt"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

// ReplyToPost publishes a reply to a status as the account.
func (c *MastodonConnector) ReplyToPost(socialAccount *connector.SocialAccount, remoteID string, remoteURL string, content string) (*connector.PostReply, error) {
	statusID, err := extractMastodonStatusID(remoteID, remoteURL)
	if err != nil {
		return nil, err
	}

	instanceURL := socialAccount.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	client := utils.NewClient(c.Logger()).PostWithJson(instanceURL+"/api/v1/statuses", map[string]any{
		"status":         content,
		"in_reply_to_id": statusID,
	})
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(socialAccount.OAuthToken)

	var created contextStatus
	client.AsJson(&created)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		if len(client.ResBody) > 0 {
			return nil, fmt.Errorf("failed to post mastodon reply: %w: %s", client.Err, string(client.ResBody))
		}
		return nil, fmt.Errorf("failed to post mastodon reply: %w", client.Err)
	}

	return created.toPostReply(), nil
}
