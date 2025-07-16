package mastodon

import (
	"github.com/jamesread/golure/pkg/redact"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"

	"context"

	"golang.org/x/oauth2"
)

type MastodonConnector struct {
	token string
	db    *db.DB

	connector.ConnectorWithWall
	connector.OAuth2Connector
	connector.OAuth2ConnectorWithClientRegistration
	connector.ConfigProvider

	utils.LogComponent
}

type Toot struct {
	Status string `json:"status"`
}

type Status struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

const CFG_MASTODON_CLIENT_ID = "mastodon.client_id"
const CFG_MASTODON_CLIENT_SECRET = "mastodon.client_secret"

func (c *MastodonConnector) IsRegistered() bool {
	clientID := c.db.GetCvarString(CFG_MASTODON_CLIENT_ID)
	clientSecret := c.db.GetCvarString(CFG_MASTODON_CLIENT_SECRET)

	return clientID != "" || clientSecret != ""
}


func (c *MastodonConnector) GetCvars() map[string]*db.Cvar {
	return map[string]*db.Cvar{
		CFG_MASTODON_CLIENT_ID: &db.Cvar{
			KeyName:      CFG_MASTODON_CLIENT_ID,
			DefaultValue: "",
			Title:        "Mastodon Client ID",
			Description:  "https://docs.joinmastodon.org/client/token/",
			Category:     "Mastodon",
			Type:         "text",
		},
		CFG_MASTODON_CLIENT_SECRET: &db.Cvar{
			KeyName:      CFG_MASTODON_CLIENT_SECRET,
			DefaultValue: "",
			Title:        "Mastodon Client Secret",
			Description:  "https://docs.joinmastodon.org/client/token/",
			Category:     "Mastodon",
			Type:         "password",
		},
	}
}

func (c *MastodonConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	res := &connector.ConfigurationCheckResult{
		Issues: []string{},
	}

	if !c.IsRegistered() {
		res.AddIssue("Mastodon client is not registered. Please register the client first.")
	}

	return res
}

func (c *MastodonConnector) GetIdentity() string {
	return "mastodon-user"
}

func (c *MastodonConnector) GetProtocol() string {
	return "mastodon"
}

func (c *MastodonConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	c.db = startup.DB
}

type AppConfig struct {
	Server string `json:"server"`
	ClientName string `json:"client_name"`
	Scopes string `json:"scopes"`
	Website string `json:"website"`
	RedirectURIs string `json:"redirect_uris"`
}

type RegistrationResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (c *MastodonConnector) RegisterClient() error {
	client := utils.NewClient()

	appConfig := &AppConfig{
		Server:       "https://mastodon.social",
		ClientName:   "japella",
		Scopes:       "read write follow",
		Website:      "https://jamesread.github.io/Japella",
		RedirectURIs: c.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL),
	}

	client.PostWithJson("https://mastodon.social/api/v1/apps", appConfig)

	if client.Err != nil {
		log.Errorf("Error: %s", client.Err)
	}

	resp := &RegistrationResponse{}

	client.AsJson(resp)

	if client.Err != nil {
		log.Errorf("Error registering Mastodon app: %v", client.Err)
		return nil
	}

	c.db.SetCvarString(CFG_MASTODON_CLIENT_ID, resp.ClientID)
	c.db.SetCvarString(CFG_MASTODON_CLIENT_SECRET, resp.ClientSecret)

	return nil
}

func (c *MastodonConnector) Start() {
	c.SetPrefix("Mastodon")
	c.Logger().Infof("Mastodon connector started")
}

func (c *MastodonConnector) PostToWall(socialAccount *connector.SocialAccount, content string) *connector.PostResult {
	res := &connector.PostResult{}

	log.Infof("Posting to wall: %s", content)

	toot := &Toot{
		Status: content,
	}

	client := utils.NewClient().PostWithJson("https://mastodon.social/api/v1/statuses", toot).WithBearerToken(socialAccount.OAuthToken)

	if client.Err != nil {
		res.Err = client.Err
		return res
	}

	postResult := &Status{}

	client.AsJson(postResult)

	if client.Err != nil {
		res.Err = client.Err
		return res
	}

	res.URL = postResult.URI

	return res
}

func (c *MastodonConnector) GetIcon() string {
	return "mdi:mastodon"
}

func (c *MastodonConnector) GetOAuth2Config() *oauth2.Config {
	c.Logger().Infof("Getting OAuth2 config for Mastodon, client_id: %v", redact.RedactString(c.db.GetCvarString(CFG_MASTODON_CLIENT_ID)))
	ep := oauth2.Endpoint{
		AuthURL:  "https://mastodon.social/oauth/authorize",
		TokenURL: "https://mastodon.social/oauth/token",
	}

	log.Infof("OAuth2 Redirect URL: %s", c.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL))

	config := &oauth2.Config{
		ClientID:     c.db.GetCvarString(CFG_MASTODON_CLIENT_ID),
		ClientSecret: c.db.GetCvarString(CFG_MASTODON_CLIENT_SECRET),
		RedirectURL:  c.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL),
		Scopes:       []string{"read", "write", "follow"},
		Endpoint:     ep,
	}

	return config
}

type VerifyCredentialsResponse struct {
	Username string `json:"username"`
}

func (c *MastodonConnector) whoami(socialAccount *db.SocialAccount) {
	client := utils.NewClient().Get("https://mastodon.social/api/v1/accounts/verify_credentials").WithBearerToken(socialAccount.OAuth2Token)

	if client.Err != nil {
		log.Errorf("Error creating request: %v", client.Err)
		return
	}

	data := &VerifyCredentialsResponse{}

	client.AsJson(&data)

	log.Infof("Whoami response: %+v", data)

	c.db.UpdateSocialAccountIdentity(socialAccount.ID, data.Username)
}

func (c *MastodonConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	c.whoami(socialAccount)
	return nil
}

func (c *MastodonConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	client := utils.NewClient()

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	config := c.GetOAuth2Config()

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))

	if err != nil {
		return err
	}

	c.Logger().Infof("OAuth2 token received: %+v", token)

	c.db.RegisterAccount(&db.SocialAccount{
		Connector:          "mastodon",
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
	})

	return err
}

