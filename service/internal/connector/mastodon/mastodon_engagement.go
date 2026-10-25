package mastodon

import (
	"fmt"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

type statusEngagement struct {
	ID              string `json:"id"`
	FavouritesCount uint32 `json:"favourites_count"`
	ReblogsCount    uint32 `json:"reblogs_count"`
	RepliesCount    uint32 `json:"replies_count"`
	QuotesCount     uint32 `json:"quotes_count"`
}

func (s statusEngagement) toPostEngagement() *connector.PostEngagement {
	return &connector.PostEngagement{
		Likes:   s.FavouritesCount,
		Reposts: s.ReblogsCount,
		Replies: s.RepliesCount,
		Quotes:  s.QuotesCount,
	}
}

// FetchPostEngagement loads favourite/boost/reply counts for a status this account published.
func (c *MastodonConnector) FetchPostEngagement(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostEngagement, error) {
	statusID, err := extractMastodonStatusID(remoteID, postURL)
	if err != nil {
		return nil, err
	}

	instanceURL := socialAccount.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	client := utils.NewClient(c.Logger()).Get(instanceURL + "/api/v1/statuses/" + statusID)
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(socialAccount.OAuthToken)

	var status statusEngagement
	client.AsJson(&status)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch mastodon status %s: %w", statusID, client.Err)
	}

	if status.ID == "" {
		return nil, fmt.Errorf("mastodon status %s not found", statusID)
	}

	return status.toPostEngagement(), nil
}
