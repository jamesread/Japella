package x

import (
	"encoding/base64"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"context"
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

	//client, req, err := utils.NewHttpClientAndGetReq("https://api.x.com/2/users/me", socialAccount.OAuth2Token)

	if client.Err != nil {
		x.Logger().Errorf("Error creating request: %v", client.Err)
		return
	}

	whoamiResult := &WhoamiResult{}

	client.AsJson(whoamiResult)

	x.db.UpdateSocialAccountIdentity(socialAccount.ID, whoamiResult.Data.Username)
}

func (x *XConnector) PostToWall(sa *connector.SocialAccount, message string) *connector.PostResult {
	res := &connector.PostResult{}

	t := &Tweet{
		Text: message,
	}

	client := utils.NewClient(x.Logger())
	client.PostWithJson("https://api.x.com/2/tweets", t).WithBearerToken(sa.OAuthToken)

	if client.Err != nil {
		res.Err = client.Err
		return res
	}

	tweetResult := &TweetResult{}

	client.AsJson(tweetResult)

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
		Scopes:       []string{"tweet.write", "users.read", "offline.access", "tweet.read"},
		Endpoint:     ep,
	}

	return config
}

func (x *XConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	x.Logger().Infof("OnRefresh called for XConnector with socialAccount: %+v", socialAccount)

	return x.RefreshToken(socialAccount)
}

func (x *XConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) (error) {
	client := utils.NewClient(x.Logger())

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	config := x.GetOAuth2Config()

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))

	if err != nil {
		return err
	}

	x.Logger().Debugf("Received token on exchange: %+v", token)

	err = x.db.RegisterAccount(&db.SocialAccount{
		Connector:          "x",
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
	})

	return err
}
