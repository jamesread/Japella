package bluesky

/**
The bluesky OAuth2 protocol is pretty wild, and needs a lot more work
than Mastodon or X (Twitter) to get working.

Useful links:
https://bsky.social/.well-known/oauth-authorization-server
*/

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
	connector.OAuth2ConnectorWithClientRegistration

	db *db.DB
}

const CFG_BSKY_CLIENT_ID = "bluesky.client_id" // notsecret
const CFG_BSKY_CLIENT_SECRET = "bluesky.client_secret" // notsecret

func (b *BlueskyConnector) IsRegistered() bool {
	clientID := b.db.GetCvarString(CFG_BSKY_CLIENT_ID)
	clientSecret := b.db.GetCvarString(CFG_BSKY_CLIENT_SECRET)

	return clientID != "" || clientSecret != ""
}

func (b *BlueskyConnector) RegisterClient() error {
	return nil
}

func (b *BlueskyConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	b.db = startup.DB
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
		AuthURL:  "https://bsky.social/oauth/par",
		TokenURL: "https://bsky.app/xrpc/com.atproto.server.createSession",
	}

	return &oauth2.Config{
		Endpoint:     ep,
		ClientID:     "japella",
		ClientSecret: "",
		Scopes:       []string{"atproto"},
		RedirectURL:  b.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL),
	}
}

func (b *BlueskyConnector) GetCvars() map[string]*db.Cvar {
	return map[string]*db.Cvar{
		"bluesky.client_id": &db.Cvar{
			KeyName:      CFG_BSKY_CLIENT_ID,
			DefaultValue: "",
			Title:        "Bluesky Client ID",
			Description:  "Client ID for Bluesky OAuth2",
			Category:     "Bluesky",
			Type:         "text",
		},
		"bluesky.client_secret": &db.Cvar{
			KeyName:      CFG_BSKY_CLIENT_SECRET,
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
