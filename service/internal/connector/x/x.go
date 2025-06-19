package x

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	log "github.com/sirupsen/logrus"

	"net/http"
)

type XConnector struct {
	config *runtimeconfig.XConfig
	connector.BaseConnector
	connector.ConnectorWithWall
	connector.OAuth2Connector

	db *db.DB

	httpClient *http.Client
}

func (x *XConnector) StartWithConfig(startup *connector.ControllerStartupConfiguration) {
	x.db = startup.DB
	x.config = startup.Config.(*runtimeconfig.XConfig)
}

func (x *XConnector) GetIdentity() string {
	return "untitled-account"
}

func (x *XConnector) GetProtocol() string {
	return "x"
}

type WhoamiResult struct {
	Data WhoamiData `json:"data"`
}

type WhoamiData struct {
	ID        string `json:"id"`
	Namespace string `json:"name"`
	Username  string `json:"username"`
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

type Tweet struct {
	Text string `json:"text"`
}

type TweetData struct {
	ID string `json:"id"`
}

type TweetResult struct {
	Data TweetData `json:"data"`
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

	x.whoami(socialAccount)

	return nil
}
