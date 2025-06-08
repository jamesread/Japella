package bluesky

import (
	"github.com/jamesread/japella/internal/connector"
)

type BlueskyConnector struct {
	connector.BaseConnector
	connector.ConnectorWithWall
}

func (b *BlueskyConnector) StartWithConfig(config any) {
}

func (b *BlueskyConnector) GetIdentity() string {
	return "?"
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
