package bluesky

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"

	"golang.org/x/oauth2"
)

type BlueskyConnector struct {
	connector.BaseConnector
	connector.OAuth2Connector
	connector.ConnectorWithWall
	connector.ConfigProvider
}

func (b *BlueskyConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {

}

func (b *BlueskyConnector) Start() {}

func (b *BlueskyConnector) GetIdentity() string {
	return "untitled-account"
}

func (b *BlueskyConnector) GetProtocol() string {
	return "bluesky"
}

func (b *BlueskyConnector) PostToWall(message string) error {
	return nil
}

func (b *BlueskyConnector) GetIcon() string {
	return "bi:bluesky"
}

func (b *BlueskyConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	return nil
}

func (b *BlueskyConnector) GetOAuth2Config() *oauth2.Config {
	ep := oauth2.Endpoint{
		AuthURL:  "https://bsky.app/auth",
		TokenURL: "https://bsky.app/xrpc/com.atproto.server.createSession",
	}

	return &oauth2.Config{
		Endpoint:     ep,
		ClientID:     "japella",
		ClientSecret: "",
		Scopes:       []string{"com.atproto.sync.subscribe", "com.atproto.repo.createRecord", "com.atproto.server.createSession"},
		RedirectURL:  "https://japella.app/oauth/callback/bluesky",
	}
}

func (b *BlueskyConnector) GetCvars() map[string]*db.Cvar {
	return map[string]*db.Cvar{
		"bluesky.client_id": &db.Cvar{
			KeyName:      "bluesky.client_id",
			DefaultValue: "",
			Title:        "Bluesky Client ID",
			Description:  "Client ID for Bluesky OAuth2",
			Category:     "Bluesky",
			Type:         "text",
		},
		"bluesky.client_secret": &db.Cvar{
			KeyName:      "bluesky.client_secret",
			DefaultValue: "",
			Title:        "Bluesky Client Secret",
			Description:  "Client Secret for Bluesky OAuth2",
			Category:     "Bluesky",
			Type:         "password",
		},
	}
}

func (b *BlueskyConnector) CheckConfiguration() error {
	return nil
}
