package x

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type XConnector struct {
	connector.BaseConnector
	connector.ConnectorWithWall
	connector.OAuth2Connector
	connector.ConfigProvider

	db *db.DB

	utils.LogComponent
}

const EXPECTED_CLIENT_ID_LENGTH = 34
const EXPECTED_CLIENT_SECRET_LENGTH = 50

const CFG_X_CLIENT_ID = "x.client_id"
const CFG_X_CLIENT_SECRET = "x.client_secret"

func (x *XConnector) GetCvars() map[string]*db.Cvar {
	return map[string]*db.Cvar{
		CFG_X_CLIENT_ID: &db.Cvar{
			KeyName:      CFG_X_CLIENT_ID,
			DefaultValue: "",
			Title:        "X Client ID",
			Description:  "X Developer Portal » App Settings » User Authentication Seetings » Edit » Keys & Tokens",
			ExternalUrl:  "https://developer.x.com/en/portal/projects-and-apps",
			DocsUrl:      "https://jamesread.github.io/Japella/connectors/x.html",
			Category:     "X",
			Type:         "text",
		},
		CFG_X_CLIENT_SECRET: &db.Cvar{
			KeyName:      CFG_X_CLIENT_SECRET,
			DefaultValue: "",
			Title:        "X Client Secret",
			Description:  "X Developer Portal » App Settings » User Authentication Seetings » Edit » Keys & Tokens",
			ExternalUrl:  "https://developer.x.com/en/portal/projects-and-apps",
			DocsUrl:      "https://jamesread.github.io/Japella/connectors/x.html",
			Category:     "X",
			Type:         "password",
		},
	}
}

func (x *XConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	res := &connector.ConfigurationCheckResult{
		Issues: []connector.ConfigurationIssue{},
	}

	clientId := x.db.GetCvarString(CFG_X_CLIENT_ID)

	if clientId == "" {
		res.AddSettingsIssue("X Client ID is not set in the database, please configure it in the settings.", CFG_X_CLIENT_ID)
	}

	if len(clientId) != EXPECTED_CLIENT_ID_LENGTH {
		res.AddSettingsIssue("X Client ID is not valid, it should be 34 characters long.", CFG_X_CLIENT_ID)
		return res
	}

	clientSecret := x.db.GetCvarString(CFG_X_CLIENT_SECRET)

	if clientSecret == "" {
		res.AddSettingsIssue("X Client Secret is not set in the database, please configure it in the settings.", CFG_X_CLIENT_SECRET)
	}

	if len(clientSecret) != EXPECTED_CLIENT_SECRET_LENGTH {
		res.AddSettingsIssue("X Client Secret is not valid, it should be 50 characters long.", CFG_X_CLIENT_SECRET)
	}

	return res
}

func (x *XConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	x.db = startup.DB
}

func (x *XConnector) Start() {
	x.SetPrefix("X")
}

func (x *XConnector) GetIdentity() string {
	return "untitled-account"
}

func (x *XConnector) GetProtocol() string {
	return "x"
}

type UpdateTokenResult struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

const xUserTweetsQuery = "max_results=20&expansions=author_id&user.fields=profile_image_url,username&tweet.fields=created_at,author_id"

func logXAPIError(logger *log.Entry, client *utils.ChainingHttpClient, msg string) {
	if len(client.ResBody) > 0 {
		logger.Errorf("%s: %v | response: %s", msg, client.Err, string(client.ResBody))
		return
	}
	logger.Errorf("%s: %v", msg, client.Err)
}

func (x *XConnector) RefreshToken(socialAccount *db.SocialAccount) error {
	// This function refreshes the OAuth2 token for a given social account
	// and then calls the whoami function to update the account's identity.
	//
	// It should really be using the OAuth2 library's token refresh capabilities,
	// but we're not using the OAuth2 client directly here, so we handle it manually.

	x.Logger().Infof("Refreshing token for X account %d (%s)", socialAccount.ID, socialAccount.Identity)

	refreshTokenArgs := make(map[string]string)
	refreshTokenArgs["refresh_token"] = socialAccount.OAuth2RefreshToken
	refreshTokenArgs["grant_type"] = "refresh_token"
	//refreshTokenArgs["client_id"] = x.db.GetCvarString(CFG_X_CLIENT_ID)

	requrl := "https://api.x.com/2/oauth2/token"
	tok := base64.StdEncoding.EncodeToString([]byte(x.db.GetCvarString(CFG_X_CLIENT_ID) + ":" + x.db.GetCvarString(CFG_X_CLIENT_SECRET)))

	client := utils.NewClient(x.Logger()).PostWithFormVars(requrl, refreshTokenArgs).WithBasicAuth(tok)

	if client.Err != nil {
		x.Logger().Errorf("Error creating request: %v", client.Err)
		return client.Err
	}

	res := &UpdateTokenResult{}

	client.AsJson(res)

	if client.Err != nil {
		logXAPIError(x.Logger(), client, "Error refreshing token")
		return client.Err
	}

	x.Logger().Debugf("Token refreshed successfully: %+v", res)

	x.db.UpdateSocialAccountToken(socialAccount.ID, res.AccessToken, res.RefreshToken, res.ExpiresIn)

	socialAccount.OAuth2Token = res.AccessToken
	x.whoami(socialAccount)

	return nil
}

func (x *XConnector) whoami(socialAccount *db.SocialAccount) {
	client := utils.NewClient(x.Logger())
	client.Get("https://api.x.com/2/users/me").WithBearerToken(socialAccount.OAuth2Token)

	if client.Err != nil {
		x.Logger().Errorf("Error creating whoami request: %v", client.Err)
		return
	}

	whoamiResult := &WhoamiResult{}

	client.AsJson(whoamiResult)

	if client.Err != nil {
		logXAPIError(x.Logger(), client, "Error parsing whoami response")
		return
	}

	if whoamiResult.Data.Username == "" {
		x.Logger().Warnf("X API returned empty username for account %d", socialAccount.ID)
		return
	}

	x.Logger().Infof("Updated X account identity to: %s", whoamiResult.Data.Username)
	x.db.UpdateSocialAccountIdentity(socialAccount.ID, whoamiResult.Data.Username)
	if whoamiResult.Data.ID != "" {
		x.db.UpdateSocialAccountDid(socialAccount.ID, whoamiResult.Data.ID)
	}
}

func (x *XConnector) lookupUserID(accessToken string) (string, error) {
	client := utils.NewClient(x.Logger())
	client.Get("https://api.x.com/2/users/me").WithBearerToken(accessToken)

	if client.Err != nil {
		return "", client.Err
	}

	whoamiResult := &WhoamiResult{}
	client.AsJson(whoamiResult)

	if client.Err != nil {
		logXAPIError(x.Logger(), client, "Error looking up X user id")
		return "", client.Err
	}

	if whoamiResult.Data.ID == "" {
		return "", fmt.Errorf("X API returned empty user id")
	}

	return whoamiResult.Data.ID, nil
}

func (x *XConnector) resolveUserID(accessToken string, cachedUserID string, accountID uint32) (string, error) {
	if cachedUserID != "" {
		return cachedUserID, nil
	}

	userID, err := x.lookupUserID(accessToken)
	if err != nil {
		return "", err
	}

	if accountID != 0 {
		x.db.UpdateSocialAccountDid(accountID, userID)
	}

	return userID, nil
}

// mediaUploadResponse is the JSON response from POST https://api.x.com/2/media/upload.
// v1-style: { "media_id": 123 } or { "media_id_string": "123" }
// v2-style: { "data": { "media_key": "..." } } or { "data": { "media_id": "..." } }
type mediaUploadResponse struct {
	MediaID       interface{}       `json:"media_id"`
	MediaIDString string            `json:"media_id_string"`
	Data          *mediaUploadData  `json:"data"`
}

type mediaUploadData struct {
	MediaKey string      `json:"media_key"`
	MediaID  interface{} `json:"media_id"`
}

func (x *XConnector) uploadMedia(path string, bearerToken string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	// X API v2 expects form field "media" with raw binary, or "media_data" with base64
	part, err := w.CreateFormFile("media", filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("write file to form: %w", err)
	}
	// Optional: set media_category for images so X treats as tweet_image
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
		_ = w.WriteField("media_category", "tweet_image")
	}
	contentType := w.FormDataContentType()
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.x.com/2/media/upload", body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", contentType)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload forbidden (403): X requires the media.write scope. Disconnect and reconnect your X account in Japella so the new permission is granted. Response: %s", string(b))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload returned %d: %s", resp.StatusCode, string(b))
	}

	var out mediaUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	// Prefer v2 data.media_key or data.media_id, then media_id_string, then media_id
	if out.Data != nil {
		if out.Data.MediaKey != "" {
			return out.Data.MediaKey, nil
		}
		if out.Data.MediaID != nil {
			return fmt.Sprint(out.Data.MediaID), nil
		}
	}
	if out.MediaIDString != "" {
		return out.MediaIDString, nil
	}
	if out.MediaID != nil {
		return fmt.Sprint(out.MediaID), nil
	}
	return "", fmt.Errorf("media upload response missing media_id / media_key (got %+v)", out)
}

// mediaKeyToTweetMediaID converts a v2 media_key (e.g. "3_2032196022764445696") to the
// format POST /2/tweets expects in media.media_ids: digits only, ^[0-9]{1,19}$
func mediaKeyToTweetMediaID(mediaKey string) string {
	if idx := strings.Index(mediaKey, "_"); idx >= 0 && idx+1 < len(mediaKey) {
		return mediaKey[idx+1:]
	}
	return mediaKey
}

func (x *XConnector) PostToWall(sa *connector.SocialAccount, message string, mediaPaths []string) *connector.PostResult {
	res := &connector.PostResult{}

	t := &Tweet{
		Text: message,
	}

	// Upload attached media and collect media_ids (X allows up to 4 images per tweet)
	if len(mediaPaths) > 0 {
		var mediaIDs []string
		for i, p := range mediaPaths {
			if i >= 4 {
				x.Logger().Warnf("X allows max 4 images per tweet, skipping remaining %d", len(mediaPaths)-4)
				break
			}
			mediaID, err := x.uploadMedia(p, sa.OAuthToken)
			if err != nil {
				x.Logger().Errorf("X media upload failed for %s: %v", p, err)
				_ = x.db.InsertTableLog(
					fmt.Sprintf("X media upload failed for account %d: %v", sa.Id, err),
					"error",
					&sa.Id,
				)
				res.Err = err
				return res
			}
			// POST /2/tweets expects media_ids to match ^[0-9]{1,19}$; v2 upload returns media_key like "3_123..."
			mediaIDs = append(mediaIDs, mediaKeyToTweetMediaID(mediaID))
		}
		if len(mediaIDs) > 0 {
			t.Media = &TweetMedia{MediaIds: mediaIDs}
		}
	}

	client := utils.NewClient(x.Logger())
	client.PostWithJson("https://api.x.com/2/tweets", t).WithBearerToken(sa.OAuthToken)

	if client.Err != nil {
		x.Logger().Errorf("Error creating POST request to X API: %v", client.Err)

		// Persist the failure to the logs table so it can be inspected from the UI
		_ = x.db.InsertTableLog(
			fmt.Sprintf("Error creating POST request to X API for account %d: %v", sa.Id, client.Err),
			"error",
			&sa.Id,
		)

		res.Err = client.Err
		return res
	}

	tweetResult := &TweetResult{}

	client.AsJson(tweetResult)

	// Check for errors after JSON parsing
	if client.Err != nil {
		x.Logger().Errorf("Error parsing X API response: %v", client.Err)
		if len(client.ResBody) > 0 {
			x.Logger().Errorf("X API response body: %s", string(client.ResBody))
		}

		// Also record this error in the logs table (commonly contains "unexpected status code: 403")
		msg := fmt.Sprintf("Error parsing X API response for account %d: %v", sa.Id, client.Err)
		if len(client.ResBody) > 0 {
			msg += " | response: " + string(client.ResBody)
		}
		_ = x.db.InsertTableLog(msg, "error", &sa.Id)

		res.Err = client.Err
		return res
	}

	// Validate that we received a valid tweet ID
	if tweetResult.Data.ID == "" {
		x.Logger().Errorf("X API returned empty tweet ID - post may have failed")

		err := fmt.Errorf("X API returned empty tweet ID")
		_ = x.db.InsertTableLog(
			fmt.Sprintf("X API returned empty tweet ID for account %d", sa.Id),
			"error",
			&sa.Id,
		)

		res.Err = err
		return res
	}

	x.Logger().Infof("Successfully posted to X, tweet ID: %s", tweetResult.Data.ID)
	res.URL = "https://x.com/user/status/" + tweetResult.Data.ID

	return res
}

func (x *XConnector) GetIcon() string {
	return "bi:twitter-x"
}

func (x *XConnector) GetOAuth2Config() *oauth2.Config {
	ep := endpoints.X

	config := &oauth2.Config{
		ClientID:     x.db.GetCvarString(CFG_X_CLIENT_ID),
		ClientSecret: x.db.GetCvarString(CFG_X_CLIENT_SECRET),
		RedirectURL:  x.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL),
		Scopes:       []string{"tweet.write", "users.read", "offline.access", "tweet.read", "media.write"},
		Endpoint:     ep,
	}

	return config
}

func (x *XConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	x.Logger().Infof("OnRefresh called for X account %d (%s)", socialAccount.ID, socialAccount.Identity)

	return x.RefreshToken(socialAccount)
}

func (x *XConnector) FetchRecentPosts(socialAccount *connector.SocialAccount) ([]*connector.FeedPost, error) {
	x.Logger().Infof("Fetching recent posts for X account %d", socialAccount.Id)

	posts, err := x.fetchRecentPostsWithToken(socialAccount.OAuthToken, socialAccount.Did, socialAccount.Id)
	if err == nil {
		return posts, nil
	}

	if !isXUnauthorized(err) {
		return posts, err
	}

	dbAccount, getErr := x.db.GetSocialAccount(socialAccount.Id)
	if getErr != nil || dbAccount == nil {
		return posts, err
	}

	if refreshErr := x.RefreshToken(dbAccount); refreshErr != nil {
		x.Logger().Errorf("Failed to refresh X token for account %d after 401: %v", socialAccount.Id, refreshErr)
		return posts, err
	}

	return x.fetchRecentPostsWithToken(dbAccount.OAuth2Token, dbAccount.Did, socialAccount.Id)
}

func (x *XConnector) fetchRecentPostsWithToken(accessToken string, cachedUserID string, accountID uint32) ([]*connector.FeedPost, error) {
	posts := make([]*connector.FeedPost, 0)

	userID, err := x.resolveUserID(accessToken, cachedUserID, accountID)
	if err != nil {
		return posts, fmt.Errorf("resolve X user id: %w", err)
	}

	client := utils.NewClient(x.Logger())
	timelineURL := fmt.Sprintf("https://api.x.com/2/users/%s/tweets?%s", userID, xUserTweetsQuery)
	client.Get(timelineURL).WithBearerToken(accessToken)

	if client.Err != nil {
		x.Logger().Errorf("Error creating request for X timeline: %v", client.Err)
		return posts, client.Err
	}

	var timelineResponse struct {
		Data []struct {
			ID        string    `json:"id"`
			Text      string    `json:"text"`
			CreatedAt time.Time `json:"created_at"`
			AuthorID  string    `json:"author_id"`
		} `json:"data"`
		Includes struct {
			Users []struct {
				ID              string `json:"id"`
				Username        string `json:"username"`
				ProfileImageURL string `json:"profile_image_url"`
			} `json:"users"`
		} `json:"includes"`
	}

	client.AsJson(&timelineResponse)

	if client.Err != nil {
		logXAPIError(x.Logger(), client, "Error parsing X timeline response")
		if isXUnauthorized(client.Err) {
			x.Logger().Warnf("X timeline request unauthorized — access token may be expired or revoked")
		}
		return posts, client.Err
	}

	usersByID := make(map[string]struct {
		Username        string
		ProfileImageURL string
	})
	for _, user := range timelineResponse.Includes.Users {
		usersByID[user.ID] = struct {
			Username        string
			ProfileImageURL string
		}{
			Username:        user.Username,
			ProfileImageURL: user.ProfileImageURL,
		}
	}

	// Convert timeline tweets to feed posts
	for _, tweet := range timelineResponse.Data {
		if tweet.AuthorID == "" {
			x.Logger().Warnf("Skipping tweet %s with empty author ID", tweet.ID)
			continue
		}

		authorName := ""
		authorAvatarURL := ""
		if user, ok := usersByID[tweet.AuthorID]; ok {
			authorName = user.Username
			authorAvatarURL = user.ProfileImageURL
		}

		feedPost := &connector.FeedPost{
			Content:         tweet.Text,
			PostedDate:      tweet.CreatedAt,
			AuthorID:        tweet.AuthorID,
			AuthorName:      authorName,
			AuthorAvatarURL: authorAvatarURL,
			RemoteURL:       "https://x.com/user/status/" + tweet.ID,
			RemoteID:        tweet.ID,
		}

		posts = append(posts, feedPost)
	}

	x.Logger().Infof("Fetched %d recent posts from X timeline", len(posts))
	return posts, nil
}

func isXUnauthorized(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "unexpected status code: 401")
}

func (x *XConnector) FetchFeedPost(socialAccount *connector.SocialAccount, remoteID string, remoteURL string) (*connector.FeedPost, error) {
	if remoteID == "" {
		return nil, fmt.Errorf("remote id is required to refetch X post")
	}

	if _, err := strconv.ParseUint(remoteID, 10, 64); err != nil {
		return nil, fmt.Errorf("invalid X tweet id %q", remoteID)
	}

	post, err := x.fetchFeedPostWithToken(socialAccount.OAuthToken, remoteID)
	if err == nil {
		return post, nil
	}

	if !isXUnauthorized(err) {
		return nil, err
	}

	dbAccount, getErr := x.db.GetSocialAccount(socialAccount.Id)
	if getErr != nil || dbAccount == nil {
		return nil, err
	}

	if refreshErr := x.RefreshToken(dbAccount); refreshErr != nil {
		return nil, err
	}

	return x.fetchFeedPostWithToken(dbAccount.OAuth2Token, remoteID)
}

func (x *XConnector) fetchFeedPostWithToken(accessToken string, remoteID string) (*connector.FeedPost, error) {
	client := utils.NewClient(x.Logger())
	client.Get("https://api.x.com/2/tweets/" + remoteID + "?expansions=author_id&user.fields=profile_image_url,username&tweet.fields=created_at,author_id").WithBearerToken(accessToken)
	if client.Err != nil {
		return nil, client.Err
	}

	var response struct {
		Data struct {
			ID        string    `json:"id"`
			Text      string    `json:"text"`
			CreatedAt time.Time `json:"created_at"`
			AuthorID  string    `json:"author_id"`
		} `json:"data"`
		Includes struct {
			Users []struct {
				ID              string `json:"id"`
				Username        string `json:"username"`
				ProfileImageURL string `json:"profile_image_url"`
			} `json:"users"`
		} `json:"includes"`
	}

	client.AsJson(&response)
	if client.Err != nil {
		return nil, fmt.Errorf("failed to fetch X tweet %s: %w", remoteID, client.Err)
	}

	if response.Data.ID == "" {
		return nil, fmt.Errorf("X tweet %s not found", remoteID)
	}

	authorName := ""
	authorAvatarURL := ""
	for _, user := range response.Includes.Users {
		if user.ID == response.Data.AuthorID {
			authorName = user.Username
			authorAvatarURL = user.ProfileImageURL
			break
		}
	}

	return &connector.FeedPost{
		Content:         response.Data.Text,
		PostedDate:      response.Data.CreatedAt,
		AuthorID:        response.Data.AuthorID,
		AuthorName:      authorName,
		AuthorAvatarURL: authorAvatarURL,
		RemoteURL:       "https://x.com/user/status/" + response.Data.ID,
		RemoteID:        response.Data.ID,
	}, nil
}

func (x *XConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	client := utils.NewClient(x.Logger())

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	config := x.GetOAuth2Config()

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))

	if err != nil {
		return err
	}

	x.Logger().Debugf("Received token on exchange: %+v", token)

	// Get identity (username) and user id before registering to match existing accounts
	identity := ""
	platformUserID := ""
	whoamiClient := utils.NewClient(x.Logger())
	whoamiClient.Get("https://api.x.com/2/users/me").WithBearerToken(token.AccessToken)
	if whoamiClient.Err == nil {
		whoamiResult := &WhoamiResult{}
		whoamiClient.AsJson(whoamiResult)
		if whoamiClient.Err == nil {
			if whoamiResult.Data.Username != "" {
				identity = whoamiResult.Data.Username
				x.Logger().Infof("Retrieved X account identity: %s", identity)
			}
			platformUserID = whoamiResult.Data.ID
		}
	}

	err = x.db.RegisterAccount(&db.SocialAccount{
		Connector:          "x",
		Identity:           identity,
		Did:                platformUserID,
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
		Active:             true,
	})

	return err
}
