package mastodon

import (
	"github.com/jamesread/golure/pkg/redact"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"

	"golang.org/x/oauth2"
)

type MastodonConnector struct {
	token  string
	db     *db.DB

	doRegistration bool
	isInert bool

	connector.ConnectorWithWall
	connector.OAuth2Connector
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
const CFG_MASTODON_REGISTER = "mastodon.register"

func (c *MastodonConnector) GetCvars() (map[string]*db.Cvar) {
	return map[string]*db.Cvar{
		CFG_MASTODON_CLIENT_ID: &db.Cvar{
			KeyName:      CFG_MASTODON_CLIENT_ID,
			DefaultValue: "",
			Title:       "Mastodon Client ID",
			Description: "https://docs.joinmastodon.org/client/token/",
			Category:    "Mastodon",
			Type:		 "text",
		},
		CFG_MASTODON_CLIENT_SECRET: &db.Cvar{
			KeyName:      CFG_MASTODON_CLIENT_SECRET,
			DefaultValue: "",
			Title:       "Mastodon Client Secret",
			Description: "https://docs.joinmastodon.org/client/token/",
			Category:    "Mastodon",
			Type:		 "password",
		},
		CFG_MASTODON_REGISTER: &db.Cvar{
			KeyName:      CFG_MASTODON_REGISTER,
			DefaultValue: "0",
			Title:       "Mastodon app registration?",
			Description: "Register a new Mastodon app on startup",
			Category:    "Mastodon",
			Type:        "bool",
		},

	}
}

func (c *MastodonConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	res := &connector.ConfigurationCheckResult{
		Issues: []string{},
	}

	if c.db.GetCvarString(CFG_MASTODON_CLIENT_ID) == "" {
		res.AddIssue("Mastodon client ID is not configured")
	}

	if c.db.GetCvarString(CFG_MASTODON_CLIENT_SECRET) == "" {
		res.AddIssue("Mastodon client secret is not configured")
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

func (c *MastodonConnector) register() {
	/**
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:     "https://mastodon.social",
		ClientName: "japella",
		Scopes:     "read write follow",
		Website:    c.config.Website,
		RedirectURIs: "http://localhost:8080/oauth2callback",
	})

	if err != nil {
		log.Errorf("Error: %s", err)
	}

	c.Logger().Infof("client-id: %v", app.ClientID)
	c.Logger().Infof("client-secret: %v", app.ClientSecret)
	c.Logger().Infof("AuthURL: %v", app.AuthURI)
	*/

	//	fmt.Println("!!! Please type your token below:")
	//	fmt.Scanln(&c.token)

	c.Logger().Infof("Token: %s", c.token)
}

func (c *MastodonConnector) Start() {
	c.SetPrefix("Mastodon")
	c.Logger().Infof("Mastodon connector started")

	if c.doRegistration {
		c.register()
	}
}

func amqpReplier() {
	/*
		amqp.ConsumeForever("mastodon-OutgoingMessage", func(d amqp.Delivery) {
			reply := msgs.OutgoingMessage{}

			amqp.Decode(d.Message.Body, &reply)

			toot := &mastodon.Toot{
				Status:     reply.Content,
				Visibility: "public",
			}

			Post(toot)
		})
	*/
}

func (c *MastodonConnector) PostToWall(socialAccount *connector.SocialAccount, content string) *connector.PostResult {
	res := &connector.PostResult{}

	log.Infof("Posting to wall: %s", content)

	if c.isInert {
		res.URL = "https://mastodon.social/@jamesread/1234567890" // Dummy URL for inert mode
		return res
	}

	toot := &Toot{
		Status: content,
	}

	client, req, err := utils.NewHttpClientAndGetReqWithJson("https://mastodon.social/api/v1/statuses", socialAccount.OAuthToken, toot)

	if err != nil {
		res.Err = err
		return res
	}

	postResult := &Status{}

	err = utils.ClientDoJson(client, req, postResult)

	if err != nil {
		res.Err = err
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
	client, req, err := utils.NewHttpClientAndGetReq("https://mastodon.social/api/v1/accounts/verify_credentials", socialAccount.OAuth2Token)

	if err != nil {
		log.Errorf("Error creating request: %v", err)
		return
	}

	data := &VerifyCredentialsResponse{}

	utils.ClientDoJson(client, req, &data)

	log.Infof("Whoami response: %+v", data)

	c.db.UpdateSocialAccountIdentity(socialAccount.ID, data.Username)
}

func (c *MastodonConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	c.whoami(socialAccount)
	return nil
}
