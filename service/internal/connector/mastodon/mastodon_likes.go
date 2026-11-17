package mastodon

import (
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
	status, err := c.postLikeAction(socialAccount, statusID, likeAction(liked))
	if err != nil {
		return nil, err
	}
	return &connector.LikeResult{
		Liked:     statusIsFavourited(status),
		LikeCount: status.FavouritesCount,
	}, nil
}

func statusIsFavourited(status *TimelineStatus) bool {
	return status.Favourited != nil && *status.Favourited
}

func likeAction(liked bool) string {
	if liked {
		return "favourite"
	}
	return "unfavourite"
}

func (c *MastodonConnector) postLikeAction(socialAccount *connector.SocialAccount, statusID, action string) (*TimelineStatus, error) {
	client := utils.NewClient(c.Logger()).PostWithJson(mastodonInstance(socialAccount)+"/api/v1/statuses/"+statusID+"/"+action, map[string]any{})
	if client.Err != nil {
		return nil, client.Err
	}
	var status TimelineStatus
	if err := c.decodeBearerJSON(client, socialAccount.OAuthToken, &status); err != nil {
		return nil, wrapUnlessRateLimited(err, "failed to %s mastodon status %s", action, statusID)
	}
	return &status, nil
}
