package mastodon

import (
	"errors"
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

func mastodonFollowState(rel *mastodonRelationship) *connector.FollowState {
	return &connector.FollowState{Following: rel.Following, Requested: rel.Requested}
}

func requireMastodonAccountID(authorID string) error {
	if !mastodonAccountIDRe.MatchString(authorID) {
		return fmt.Errorf("invalid mastodon account id %q", authorID)
	}
	return nil
}

func (c *MastodonConnector) decodeBearerJSON(client *utils.ChainingHttpClient, token string, dest any) error {
	client.WithBearerToken(token)
	client.AsJson(dest)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return rl
	}
	return client.Err
}

func wrapUnlessRateLimited(err error, format string, args ...any) error {
	if _, ok := errors.AsType[*connector.RateLimitedError](err); ok {
		return err
	}
	args = append(append([]any{}, args...), err)
	return fmt.Errorf(format+": %w", args...)
}

// GetFollowState reads the account's relationship with an author. Account ids
// are local to the account's instance, which is where feed and search
// results come from.
func (c *MastodonConnector) GetFollowState(sa *connector.SocialAccount, authorID string) (*connector.FollowState, error) {
	if err := requireMastodonAccountID(authorID); err != nil {
		return nil, err
	}
	rels, err := c.fetchRelationships(sa, authorID)
	if err != nil {
		return nil, err
	}
	return relationshipForAuthor(rels, authorID), nil
}

func (c *MastodonConnector) fetchRelationships(sa *connector.SocialAccount, authorID string) ([]mastodonRelationship, error) {
	client := utils.NewClient(c.Logger()).Get(mastodonInstance(sa) + "/api/v1/accounts/relationships?id[]=" + url.QueryEscape(authorID))
	if client.Err != nil {
		return nil, client.Err
	}
	var rels []mastodonRelationship
	if err := c.decodeBearerJSON(client, sa.OAuthToken, &rels); err != nil {
		return nil, wrapUnlessRateLimited(err, "failed to read relationship with mastodon account %s", authorID)
	}
	return rels, nil
}

func relationshipForAuthor(rels []mastodonRelationship, authorID string) *connector.FollowState {
	for i := range rels {
		if rels[i].ID == authorID {
			return mastodonFollowState(&rels[i])
		}
	}
	return &connector.FollowState{}
}

// SetFollowing follows or unfollows an author as the account.
func (c *MastodonConnector) SetFollowing(sa *connector.SocialAccount, authorID string, follow bool) (*connector.FollowState, error) {
	if err := requireMastodonAccountID(authorID); err != nil {
		return nil, err
	}
	rel, err := c.postFollowAction(sa, authorID, followAction(follow))
	if err != nil {
		return nil, err
	}
	return mastodonFollowState(rel), nil
}

func followAction(follow bool) string {
	if follow {
		return "follow"
	}
	return "unfollow"
}

func (c *MastodonConnector) postFollowAction(sa *connector.SocialAccount, authorID, action string) (*mastodonRelationship, error) {
	client := utils.NewClient(c.Logger()).PostWithJson(mastodonInstance(sa)+"/api/v1/accounts/"+authorID+"/"+action, map[string]any{})
	if client.Err != nil {
		return nil, client.Err
	}
	var rel mastodonRelationship
	if err := c.decodeBearerJSON(client, sa.OAuthToken, &rel); err != nil {
		return nil, wrapUnlessRateLimited(err, "failed to %s mastodon account %s", action, authorID)
	}
	return &rel, nil
}
