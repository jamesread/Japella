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

// ConnectorWithDisplayName is an optional interface for connectors that support custom display names
type ConnectorWithDisplayName interface {
	GetDisplayName() string
}

type ConnectorWithChannels interface {
	PostToChannel(channelId string, message string)
}

type ConnectorWithWall interface {
	BaseConnector

	PostToWall(sa *SocialAccount, message string, mediaURLs []string) *PostResult
	FetchRecentPosts(sa *SocialAccount) ([]*FeedPost, error)
}

// ChatBot represents a chatbot instance from a connector
type ChatBot struct {
	Connector          string // The connector protocol (e.g., "telegram", "discord")
	Name               string // Display name of the bot
	Identity           string // Bot identity/username
	Icon               string // Icon identifier
	IsRunning          bool   // Whether the bot is currently running
	StatusMessage      string // Diagnostic message explaining why the bot is stopped or its current status
	ErrorMessage       string // Error message if the bot failed to start
	ProtocolDisplayName string // Protocol-specific display name (e.g., Telegram bot's FirstName)
}

// ConnectorWithChatBot is an interface for connectors that provide chatbot functionality
type ConnectorWithChatBot interface {
	BaseConnector
	GetChatBot() *ChatBot
}

// BotChannel represents a channel/chat that a bot is in
type BotChannel struct {
	ID       string // Channel/chat ID
	Title    string // Channel/chat title
	Type     string // Channel type (e.g., "channel", "group", "private")
	Username string // Username if available
}

// ConnectorWithChannelsInfo is an interface for connectors that can provide channel information
type ConnectorWithChannelsInfo interface {
	BaseConnector
	GetChannels() []*BotChannel
}

// ConnectorWithHooks is an interface for connectors that support incoming message hooks
type ConnectorWithHooks interface {
	BaseConnector
	GetHooks() []*IncomingMessageHook
	SetHooks(hooks []*IncomingMessageHook) error
}

// IncomingMessageHook represents a webhook to call when messages are received
type IncomingMessageHook struct {
	URL     string
	Enabled bool
}

// ConnectorWithYamlConfig is a marker interface for connectors that use YAML configuration (as opposed to OAuth)
type ConnectorWithYamlConfig interface {
	BaseConnector
}

// UnregisteredConnector represents a connector type that exists but isn't currently started
type UnregisteredConnector struct {
	Protocol string // The connector protocol (e.g., "telegram", "discord")
	Name     string // Display name
	Icon     string // Icon identifier
}

type FeedPost struct {
	Content            string
	PostedDate         time.Time
	AuthorID           uint32
	AuthorName         string
	RemoteURL          string
	RemoteID           string
	PreviewURL         string
	PreviewTitle       string
	PreviewDescription string
	PreviewImageURL    string
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
