package mastodon

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

var mastodonAccountIDRe = regexp.MustCompile(`^[0-9]{1,32}$`)

// mastodonRelationship is the part of a Mastodon Relationship we use.
type mastodonRelationship struct {
	ID        string `json:"id"`
	Following bool   `json:"following"`
	Requested bool   `json:"requested"`
}

func mastodonInstance(sa *connector.SocialAccount) string {
	if sa.Homeserver == "" {
		return "https://mastodon.social"
	}
	return sa.Homeserver
}

func mastodonFollowState(rel *mastodonRelationship) *connector.FollowState {
	return &connector.FollowState{Following: rel.Following, Requested: rel.Requested}
}

// GetFollowState reads the account's relationship with an author. Account ids
// are local to the account's instance, which is where feed and search
// results come from.
func (c *MastodonConnector) GetFollowState(sa *connector.SocialAccount, authorID string) (*connector.FollowState, error) {
	if !mastodonAccountIDRe.MatchString(authorID) {
		return nil, fmt.Errorf("invalid mastodon account id %q", authorID)
	}

	client := utils.NewClient(c.Logger()).Get(mastodonInstance(sa) + "/api/v1/accounts/relationships?id[]=" + url.QueryEscape(authorID))
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(sa.OAuthToken)

	var rels []mastodonRelationship
	client.AsJson(&rels)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to read relationship with mastodon account %s: %w", authorID, client.Err)
	}
	for i := range rels {
		if rels[i].ID == authorID {
			return mastodonFollowState(&rels[i]), nil
		}
	}
	return &connector.FollowState{}, nil
}

// SetFollowing follows or unfollows an author as the account.
func (c *MastodonConnector) SetFollowing(sa *connector.SocialAccount, authorID string, follow bool) (*connector.FollowState, error) {
	if !mastodonAccountIDRe.MatchString(authorID) {
		return nil, fmt.Errorf("invalid mastodon account id %q", authorID)
	}

	action := "follow"
	if !follow {
		action = "unfollow"
	}

	client := utils.NewClient(c.Logger()).PostWithJson(mastodonInstance(sa)+"/api/v1/accounts/"+authorID+"/"+action, map[string]any{})
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(sa.OAuthToken)

	var rel mastodonRelationship
	client.AsJson(&rel)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to %s mastodon account %s: %w", action, authorID, client.Err)
	}
	return mastodonFollowState(&rel), nil
}
