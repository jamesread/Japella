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
			Description:  "X Developer Portal &raquo; App Settings &raquo; User Authentication Seetings &raquo; Edit &raquo; Keys &amp; Tokens",
			ExternalUrl:  "https://developer.x.com/en/portal/projects-and-apps",
			DocsUrl:      "https://jamesread.github.io/Japella/connectors/x.html",
			Category:     "X",
			Type:         "text",
		},
		CFG_X_CLIENT_SECRET: &db.Cvar{
			KeyName:      CFG_X_CLIENT_SECRET,
			DefaultValue: "",
			Title:        "X Client Secret",
			Description:  "X Developer Portal &raquo; App Settings &raquo; User Authentication Seetings &raquo; Edit &raquo; Keys &amp; Tokens",
			ExternalUrl:  "https://developer.x.com/en/portal/projects-and-apps",
			DocsUrl:      "https://jamesread.github.io/Japella/connectors/x.html",
			Category:     "X",
			Type:         "password",
		},
	}
}

func (x *XConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	res := &connector.ConfigurationCheckResult{
		Issues: []string{},
	}

	clientId := x.db.GetCvarString(CFG_X_CLIENT_ID)

	if clientId == "" {
		res.AddIssue("X Client ID is not set in the database, please configure it in the settings.")
	}

	if len(clientId) != EXPECTED_CLIENT_ID_LENGTH {
		res.AddIssue("X Client ID is not valid, it should be 34 characters long.")
		return res
	}

	clientSecret := x.db.GetCvarString(CFG_X_CLIENT_SECRET)

	if clientSecret == "" {
		res.AddIssue("X Client Secret is not set in the database, please configure it in the settings.")
	}

	if len(clientSecret) != EXPECTED_CLIENT_SECRET_LENGTH {
		res.AddIssue("X Client Secret is not valid, it should be 50 characters long.")
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

func (x *XConnector) RefreshToken(socialAccount *db.SocialAccount) error {
	// This function refreshes the OAuth2 token for a given social account
	// and then calls the whoami function to update the account's identity.
	//
	// It should really be using the OAuth2 library's token refresh capabilities,
	// but we're not using the OAuth2 client directly here, so we handle it manually.

	x.Logger().Infof("Refreshing token for XConnector with socialAccount: %+v", socialAccount)

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
		x.Logger().Errorf("Error refreshing token: %v", client.Err)
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
		x.Logger().Errorf("Error parsing whoami response: %v", client.Err)
		return
	}

	if whoamiResult.Data.Username == "" {
		x.Logger().Warnf("X API returned empty username for account %d", socialAccount.ID)
		return
	}

	x.Logger().Infof("Updated X account identity to: %s", whoamiResult.Data.Username)
	x.db.UpdateSocialAccountIdentity(socialAccount.ID, whoamiResult.Data.Username)
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
	x.Logger().Infof("OnRefresh called for XConnector with socialAccount: %+v", socialAccount)

	return x.RefreshToken(socialAccount)
}

func (x *XConnector) FetchRecentPosts(socialAccount *connector.SocialAccount) ([]*connector.FeedPost, error) {
	x.Logger().Infof("Fetching recent posts for X account %d", socialAccount.Id)

	posts := make([]*connector.FeedPost, 0)

	// Get user's timeline (recent tweets)
	client := utils.NewClient(x.Logger())
	client.Get("https://api.x.com/2/users/me/tweets?max_results=20").WithBearerToken(socialAccount.OAuthToken)

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
	}

	client.AsJson(&timelineResponse)

	if client.Err != nil {
		x.Logger().Errorf("Error parsing X timeline response: %v", client.Err)
		return posts, client.Err
	}

	// Convert timeline tweets to feed posts
	for _, tweet := range timelineResponse.Data {
		// Parse author ID as uint32
		authorID, err := strconv.ParseUint(tweet.AuthorID, 10, 32)
		if err != nil {
			x.Logger().Warnf("Failed to parse author ID %s: %v", tweet.AuthorID, err)
			continue
		}

		feedPost := &connector.FeedPost{
			Content:    tweet.Text,
			PostedDate: tweet.CreatedAt,
			AuthorID:   uint32(authorID),
			RemoteURL:  "https://x.com/user/status/" + tweet.ID,
			RemoteID:   tweet.ID,
		}

		posts = append(posts, feedPost)
	}

	x.Logger().Infof("Fetched %d recent posts from X timeline", len(posts))
	return posts, nil
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

	// Get identity (username) before registering to match existing accounts
	identity := ""
	whoamiClient := utils.NewClient(x.Logger())
	whoamiClient.Get("https://api.x.com/2/users/me").WithBearerToken(token.AccessToken)
	if whoamiClient.Err == nil {
		whoamiResult := &WhoamiResult{}
		whoamiClient.AsJson(whoamiResult)
		if whoamiClient.Err == nil && whoamiResult.Data.Username != "" {
			identity = whoamiResult.Data.Username
			x.Logger().Infof("Retrieved X account identity: %s", identity)
		}
	}

	err = x.db.RegisterAccount(&db.SocialAccount{
		Connector:          "x",
		Identity:           identity,
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
		Active:             true,
	})

	return err
}
