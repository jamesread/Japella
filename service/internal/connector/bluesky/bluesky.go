package bluesky

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

type BlueskyConnector struct {
	connector.BaseConnector
	connector.ConnectorWithWall
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
