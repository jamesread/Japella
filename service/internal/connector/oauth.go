package connector

import (
	"golang.org/x/oauth2"
)

type OAuth2Connector interface {
	BaseConnector

	GetOAuth2Config() *oauth2.Config
	OnOAuth2Callback(code string, verifier string, headers map[string]string) (error)
}

// eg: rfc7591
type OAuth2ConnectorWithClientRegistration interface {
	RegisterClient() error
	IsRegistered() bool
}
