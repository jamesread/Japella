package mastodon

import (
	"context"
	"fmt"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"github.com/mattn/go-mastodon"
	log "github.com/sirupsen/logrus"
	"net/url"
)

var client *mastodon.Client

type MastodonConnector struct {
	token     string
	libconfig *mastodon.Config
	config    *runtimeconfig.MastodonConfig

	connector.ConnectorWithWall
	utils.LogComponent
}

func (adaptor *MastodonConnector) GetIdentity() string {
	return "mastodon-user"
}

func (adaptor *MastodonConnector) GetProtocol() string {
	return "mastodon"
}

func (adaptor *MastodonConnector) StartWithURL(url *url.URL) {
	/*
		adaptor.config = runtimeconfig.Get().Connectors.Mastodon

		adaptor.config.ClientID := url.Query().Get("clientId")
		adaptor.config.ClientSecret := url.Query().Get("clientSecret")
		adaptor.config.Register := url.Query().Get("register") == "true"
	*/

	adaptor.Start()
}

func (adaptor *MastodonConnector) StartWithConfig(rawconfig any) {
	config, _ := rawconfig.(*runtimeconfig.MastodonConfig)

	adaptor.config = config
	adaptor.Start()
}

func (adaptor *MastodonConnector) register() {
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:     "https://mastodon.social",
		ClientName: "japella",
		Scopes:     "read write follow",
		Website:    adaptor.config.Website,
	})

	if err != nil {
		log.Errorf("Error: %s", err)
	}

	adaptor.Logger().Infof("client-id: %v", app.ClientID)
	adaptor.Logger().Infof("client-secret: %v", app.ClientSecret)
	adaptor.Logger().Infof("AuthURL: %v", app.AuthURI)

	fmt.Println("!!! Please type your token below:")
	fmt.Scanln(&adaptor.token)

	adaptor.Logger().Infof("Token: %s", adaptor.token)

	adaptor.libconfig = &mastodon.Config{
		Server:       "https://mastodon.social",
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		AccessToken:  adaptor.token,
	}
}

func (adaptor *MastodonConnector) Start() {
	adaptor.SetPrefix("Mastodon")
	adaptor.Logger().Infof("Mastodon connector started")

	if adaptor.config.Register {
		adaptor.register()
	} else {
		adaptor.libconfig = &mastodon.Config{
			Server:       "https://mastodon.social",
			ClientID:     adaptor.config.ClientId,
			ClientSecret: adaptor.config.ClientSecret,
			AccessToken:  adaptor.config.Token,
		}
		adaptor.token = adaptor.config.Token
	}

	log.Infof("libconfig: %+v", adaptor.libconfig)

	client = mastodon.NewClient(adaptor.libconfig)

	err := client.AuthenticateToken(context.Background(), adaptor.token, "urn:ietf:wg:oauth:2.0:oob")

	if err != nil {
		adaptor.Logger().Errorf("Error: %s", err)
	}

	account, err := client.GetAccountCurrentUser(context.Background())

	if err != nil {
		adaptor.Logger().Errorf("Error: %s", err)
	}

	log.Infof("Account: %v", account)

	if runtimeconfig.Get().Amqp.Enabled {
		go Replier()
	}
}

func Replier() {
	amqp.ConsumeForever("mastodon-OutgoingMessage", func(d amqp.Delivery) {
		reply := msgs.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		toot := &mastodon.Toot{
			Status:     reply.Content,
			Visibility: "public",
		}

		Post(toot)
	})
}

func (adaptor *MastodonConnector) PostToWall(content string) error {
	toot := &mastodon.Toot{
		Status:     content,
		Visibility: "public",
	}

	log.Infof("Posting to wallx: %s", content)

	_, err := client.PostStatus(context.Background(), toot)

	if err != nil {
		log.Errorf("Error posting to wall: %v", err)
		return fmt.Errorf("failed to post to wall: %w", err)
	}

	return nil
}

func Post(toot *mastodon.Toot) {
	log.Infof("Post: %v", toot)

	//	post, err := c.PostStatus(context.Background(), toot)

	// log.Errorf("Error: %s", err)
}
