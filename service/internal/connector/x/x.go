package x

import (
	"encoding/base64"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	log "github.com/sirupsen/logrus"
)

type XConnector struct {
	config *runtimeconfig.XConfig
	connector.BaseConnector
	connector.ConnectorWithWall
	connector.OAuth2Connector

	db *db.DB
}

func (x *XConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	x.db = startup.DB
	x.config = startup.Config.(*runtimeconfig.XConfig)
}

func (x *XConnector) Start() {}

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

	log.Infof("Refreshing token for XConnector with socialAccount: %+v", socialAccount)

	refreshTokenArgs := make(map[string]string)
	refreshTokenArgs["refresh_token"] = socialAccount.OAuth2RefreshToken
	refreshTokenArgs["grant_type"] = "refresh_token"
	refreshTokenArgs["client_id"] = x.config.ClientID

	requrl := "https://api.x.com/2/oauth2/token"
	tok := base64.StdEncoding.EncodeToString([]byte(x.config.ClientID + ":" + x.config.ClientSecret))

	client, req, err := utils.NewHttpClientAndGetReqWithUrlEncodedMap(requrl, tok, refreshTokenArgs)

	if err != nil {
		log.Errorf("Error creating request: %v", err)
		return err
	}

	res := &UpdateTokenResult{}

	err = utils.ClientDoJson(client, req, res)

	if err != nil {
		log.Errorf("Error refreshing token: %v", err)
		return err
	}

	log.Debugf("Token refreshed successfully: %+v", res)

	x.db.UpdateSocialAccountToken(socialAccount.ID, res.AccessToken, res.RefreshToken, res.ExpiresIn)

	socialAccount.OAuth2Token = res.AccessToken
	x.whoami(socialAccount)

	return nil
}

func (x *XConnector) whoami(socialAccount *db.SocialAccount) {
	client, req, err := utils.NewHttpClientAndGetReq("https://api.x.com/2/users/me", socialAccount.OAuth2Token)

	if err != nil {
		log.Errorf("Error creating request: %v", err)
		return
	}

	whoamiResult := &WhoamiResult{}
	utils.ClientDoJson(client, req, whoamiResult)

	x.db.UpdateSocialAccountIdentity(socialAccount.ID, whoamiResult.Data.Username)
}

func (x *XConnector) PostToWall(sa *connector.SocialAccount, message string) *connector.PostResult {
	res := &connector.PostResult{}

	t := &Tweet{
		Text: message,
	}

	client, req, err := utils.NewHttpClientAndGetReqWithJson("https://api.x.com/2/tweets", sa.OAuthToken, t)

	if err != nil {
		res.Err = err
		return res
	}

	tweetResult := &TweetResult{}

	utils.ClientDoJson(client, req, tweetResult)

	res.URL = "https://x.com/user/status/" + tweetResult.Data.ID

	return res
}

func (x *XConnector) GetIcon() string {
	return "bi:twitter-x"
}

func (x *XConnector) GetOAuth2Config() *oauth2.Config {
	ep := endpoints.X

	config := &oauth2.Config{
		ClientID:     x.config.ClientID,
		ClientSecret: x.config.ClientSecret,
		RedirectURL:  "http://localhost:8080/oauth2callback",
		Scopes:       []string{"tweet.write", "users.read", "offline.access", "tweet.read"},
		Endpoint:     ep,
	}

	return config
}

func (x *XConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	log.Infof("OnRefresh called for XConnector with socialAccount: %+v", socialAccount)

	x.RefreshToken(socialAccount)

	return nil
}
