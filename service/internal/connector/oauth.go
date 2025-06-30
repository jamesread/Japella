package connector

import (
	"golang.org/x/oauth2"
)

type OAuth2Connector interface {
	BaseConnector

	GetOAuth2Config() *oauth2.Config
}

// eg: rfc7591
type OAuth2ConnectorWithClientRegistration interface {
	RegisterClient() error
	IsRegistered() bool
}
