package connector

import (
	"github.com/jamesread/japella/internal/db"
)

type BaseConnector interface {
	StartWithConfig(*ControllerStartupConfiguration)
	GetIdentity() string
	GetProtocol() string
	GetIcon() string
	OnRefresh(socialAccount *db.SocialAccount) error
}

type ConnectorWithChannels interface {
	PostToChannel(channelId string, message string)
}

type ConnectorWithWall interface {
	BaseConnector

	PostToWall(sa *SocialAccount, message string) *PostResult
}

type SocialAccount struct {
	Id         uint32
	Connector  string
	Identity   string
	OAuthToken string
}

type ControllerStartupConfiguration struct {
	DB     *db.DB
	Config any
}

type PostResult struct {
	Err error
	URL string
}
