package connector

import (
	"time"

	"github.com/jamesread/japella/internal/db"
)

type BaseConnector interface {
	SetStartupConfiguration(*ControllerStartupConfiguration)
	Start()
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
	FetchRecentPosts(sa *SocialAccount) ([]*FeedPost, error)
}

type FeedPost struct {
	Content          string
	PostedDate       time.Time
	AuthorID         uint32
	AuthorName       string
	RemoteURL        string
	RemoteID         string
	PreviewURL       string
	PreviewTitle     string
	PreviewDescription string
	PreviewImageURL  string
}

type SocialAccount struct {
	Id         uint32
	Connector  string
	Identity   string
	Did        string
	OAuthToken string
	Homeserver string
}

type ControllerStartupConfiguration struct {
	DB     *db.DB
	Config any
}

type PostResult struct {
	Err error
	URL string
}
