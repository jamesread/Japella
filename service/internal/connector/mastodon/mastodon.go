package mastodon

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"

	"golang.org/x/oauth2"
)

type MastodonConnector struct {
	token  string
	db     *db.DB
	config *runtimeconfig.MastodonConfig

	connector.ConnectorWithWall
	connector.OAuth2Connector

	utils.LogComponent
}

type Toot struct {
	Status string `json:"status"`
}

type Status struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

func (c *MastodonConnector) GetIdentity() string {
	return "mastodon-user"
}

func (c *MastodonConnector) GetProtocol() string {
	return "mastodon"
}

func (c *MastodonConnector) StartWithConfig(startup *connector.ControllerStartupConfiguration) {
	c.db = startup.DB

	config, _ := startup.Config.(*runtimeconfig.MastodonConfig)

	c.config = config
	c.Start()
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

	if c.config.Register {
		c.register()
	}

	if runtimeconfig.Get().Amqp.Enabled {
		go amqpReplier()
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

	if c.config.Inert {
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
	ep := oauth2.Endpoint{
		AuthURL:  "https://mastodon.social/oauth/authorize",
		TokenURL: "https://mastodon.social/oauth/token",
	}

	config := &oauth2.Config{
		ClientID:     c.config.ClientId,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  "http://localhost:8080/oauth2callback",
		Scopes:       []string{"read", "write", "follow"},
		Endpoint:     ep,
	}

	return config
}

type VerifyCredentialsResponse struct {
	Username string `json:"username"`
}

func (c *MastodonConnector) whoami(socialAccount *connector.SocialAccount) {
	client, req, err := utils.NewHttpClientAndGetReq("https://mastodon.social/api/v1/accounts/verify_credentials", socialAccount.OAuthToken)

	if err != nil {
		log.Errorf("Error creating request: %v", err)
		return
	}

	data := &VerifyCredentialsResponse{}

	utils.ClientDoJson(client, req, &data)

	log.Infof("Whoami response: %+v", data)

	c.db.UpdateSocialAccountIdentity(socialAccount.Id, data.Username)
}

func (c *MastodonConnector) OnRefresh(socialAccount *connector.SocialAccount) error {
	c.whoami(socialAccount)
	return nil
}
