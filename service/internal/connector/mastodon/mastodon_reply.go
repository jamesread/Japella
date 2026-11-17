package mastodon

import (
	"errors"
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
	created, err := c.createReplyStatus(socialAccount, statusID, content)
	if err != nil {
		return nil, err
	}
	return created.toPostReply(), nil
}

func (c *MastodonConnector) createReplyStatus(socialAccount *connector.SocialAccount, statusID, content string) (*contextStatus, error) {
	client := utils.NewClient(c.Logger()).PostWithJson(mastodonInstance(socialAccount)+"/api/v1/statuses", map[string]any{
		"status":         content,
		"in_reply_to_id": statusID,
	})
	if client.Err != nil {
		return nil, client.Err
	}
	var created contextStatus
	if err := c.decodeBearerJSON(client, socialAccount.OAuthToken, &created); err != nil {
		return nil, replyCreateError(err, client.ResBody)
	}
	return &created, nil
}

func replyCreateError(err error, resBody []byte) error {
	if _, ok := errors.AsType[*connector.RateLimitedError](err); ok {
		return err
	}
	if len(resBody) > 0 {
		return fmt.Errorf("failed to post mastodon reply: %w: %s", err, string(resBody))
	}
	return fmt.Errorf("failed to post mastodon reply: %w", err)
}
