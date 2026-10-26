package mastodon

import (
	"fmt"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

// SetPostLiked favourites or unfavourites a status as the account. Status ids
// are local to the account's instance, which is where feed and search
// results come from.
func (c *MastodonConnector) SetPostLiked(socialAccount *connector.SocialAccount, remoteID string, remoteURL string, liked bool) (*connector.LikeResult, error) {
	statusID, err := extractMastodonStatusID(remoteID, remoteURL)
	if err != nil {
		return nil, err
	}

	instanceURL := socialAccount.Homeserver
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}

	action := "favourite"
	if !liked {
		action = "unfavourite"
	}

	client := utils.NewClient(c.Logger()).PostWithJson(instanceURL+"/api/v1/statuses/"+statusID+"/"+action, map[string]any{})
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(socialAccount.OAuthToken)

	var status TimelineStatus
	client.AsJson(&status)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to %s mastodon status %s: %w", action, statusID, client.Err)
	}

	return &connector.LikeResult{
		Liked:     status.Favourited != nil && *status.Favourited,
		LikeCount: status.FavouritesCount,
	}, nil
}
