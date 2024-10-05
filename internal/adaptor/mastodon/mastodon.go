package mastodon

import (
	"context"
	log "github.com/sirupsen/logrus"
	pb "github.com/jamesread/japella/gen/protobuf"
	"fmt"
	"github.com/mattn/go-mastodon"
	"github.com/jamesread/japella/internal/amqp"
)

type MastodonConfig struct {
	Register bool
	AppId string
	ClientId string
	Website string

	Inert bool
}

var c *mastodon.Client

type mastodonAdaptor struct {
	config *MastodonConfig
	libconfig *mastodon.Config
	token string
}

func New(mastodonConfig *MastodonConfig) *mastodonAdaptor {
	return &mastodonAdaptor{
		config: mastodonConfig,
	}
}

func (adaptor *mastodonAdaptor) register() {
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:       "https://mastodon.social",
		ClientName: "japella",
		Scopes: "read write follow",
		Website: adaptor.config.Website,
	})

	log.Errorf("Error: %s", err)

	log.Infof("client-id: %v", app.ClientID)
	log.Infof("client-secret: %v", app.ClientSecret)
	log.Infof("AuthURL: %v", app.AuthURI)

	fmt.Scanln(&adaptor.token)

	adaptor.libconfig = &mastodon.Config{
		Server:       "https://mastodon.social",
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		AccessToken:  adaptor.token,
	}
}

func (adaptor *mastodonAdaptor) Start() {
	if (adaptor.config.Register) {
		adaptor.register()
	}

	client := mastodon.NewClient(adaptor.libconfig)

	err := client.AuthenticateToken(context.Background(), adaptor.token, "urn:ietf:wg:oauth:2.0:oob")

	log.Errorf("Error: %s", err)

	account, err := c.GetAccountCurrentUser(context.Background())

	log.Errorf("Error: %s", err)

	log.Infof("Account: %v", account)

	go Replier()
}

func Replier() {
	amqp.ConsumeForever("mastodon-OutgoingMessage", func(d amqp.Delivery) {
		reply := pb.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		toot := &mastodon.Toot{
			Status:     reply.Content,
			Visibility: "public",
		}

		Post(toot)
	})
}

func Post(toot *mastodon.Toot) {
	log.Infof("Post: %v", toot)

//	post, err := c.PostStatus(context.Background(), toot)

//	log.Errorf("Error: %s", err)
}
