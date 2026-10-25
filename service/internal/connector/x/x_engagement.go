package x

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
)

var xStatusIDPattern = regexp.MustCompile(`/status/(\d+)`)

func extractXTweetID(remoteID, postURL string) (string, error) {
	if remoteID != "" {
		if _, err := strconv.ParseUint(remoteID, 10, 64); err == nil {
			return remoteID, nil
		}
	}

	if match := xStatusIDPattern.FindStringSubmatch(postURL); len(match) == 2 {
		return match[1], nil
	}

	return "", fmt.Errorf("cannot determine X tweet id from remote_id=%q post_url=%q", remoteID, postURL)
}

type xPublicMetrics struct {
	LikeCount       uint32 `json:"like_count"`
	RetweetCount    uint32 `json:"retweet_count"`
	ReplyCount      uint32 `json:"reply_count"`
	QuoteCount      uint32 `json:"quote_count"`
	ImpressionCount uint32 `json:"impression_count"`
}

func (m xPublicMetrics) toPostEngagement() *connector.PostEngagement {
	return &connector.PostEngagement{
		Likes:       m.LikeCount,
		Reposts:     m.RetweetCount,
		Replies:     m.ReplyCount,
		Quotes:      m.QuoteCount,
		Impressions: m.ImpressionCount,
	}
}

// FetchPostEngagement loads public_metrics for a tweet this account published.
func (x *XConnector) FetchPostEngagement(socialAccount *connector.SocialAccount, remoteID string, postURL string) (*connector.PostEngagement, error) {
	tweetID, err := extractXTweetID(remoteID, postURL)
	if err != nil {
		return nil, err
	}

	engagement, err := x.fetchPostEngagementWithToken(socialAccount.OAuthToken, tweetID)
	if err == nil || !isXUnauthorized(err) {
		return engagement, err
	}

	dbAccount, getErr := x.db.GetSocialAccount(socialAccount.Id)
	if getErr != nil || dbAccount == nil {
		return nil, err
	}

	if refreshErr := x.RefreshToken(dbAccount); refreshErr != nil {
		return nil, err
	}

	return x.fetchPostEngagementWithToken(dbAccount.OAuth2Token, tweetID)
}

func (x *XConnector) fetchPostEngagementWithToken(accessToken string, tweetID string) (*connector.PostEngagement, error) {
	client := utils.NewClient(x.Logger()).Get("https://api.x.com/2/tweets/" + tweetID + "?tweet.fields=public_metrics")
	if client.Err != nil {
		return nil, client.Err
	}
	client.WithBearerToken(accessToken)

	var response struct {
		Data struct {
			ID            string         `json:"id"`
			PublicMetrics xPublicMetrics `json:"public_metrics"`
		} `json:"data"`
	}

	client.AsJson(&response)
	if rl := connector.RateLimitedFromResponse(client.Res); rl != nil {
		return nil, rl
	}
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch X tweet %s: %w", tweetID, client.Err)
	}

	if response.Data.ID == "" {
		return nil, fmt.Errorf("X tweet %s not found", tweetID)
	}

	return response.Data.PublicMetrics.toPostEngagement(), nil
}
