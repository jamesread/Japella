package connector

import (
	"context"
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

// ConnectorWithFeedPostRefetch can reload a single feed post from the remote platform.
type ConnectorWithFeedPostRefetch interface {
	FetchFeedPost(sa *SocialAccount, remoteID string, remoteURL string) (*FeedPost, error)
}

// ChatBot represents a chatbot instance from a connector
type ChatBot struct {
	BotID               string // Immutable instance id (e.g. botA)
	Connector          string // The connector protocol (e.g., "telegram", "discord")
	Name               string // Display name of the bot
	Identity           string // Bot identity/username on the platform (after connect)
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

// ConnectorWithChatBotControl is implemented by chat bot connectors that can be started and stopped at runtime.
type ConnectorWithChatBotControl interface {
	ConnectorWithChatBot
	StartChatBot() error
	StopChatBot() error
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
	Protocol         string // The connector protocol (e.g., "telegram", "discord")
	Name             string // Display name
	Icon             string // Icon identifier
	NotStartedReason string // Why the connector is not started (e.g., "Requires IsPubliclyAccessible", "Requires YAML configuration")
}

type FeedPost struct {
	Content                 string
	PostedDate              time.Time
	AuthorID                string
	AuthorName              string
	AuthorAvatarURL         string
	OriginalAuthorID        string
	OriginalAuthorName      string
	OriginalAuthorAvatarURL string
	RemoteURL               string
	RemoteID                string
	PreviewURL              string
	PreviewTitle            string
	PreviewDescription      string
	PreviewImageURL         string
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

// ChatBotStartupConfig is passed when registering a DB-backed chat bot instance.
type ChatBotStartupConfig struct {
	Protocol    string
	BotID       string
	DisplayName string
}

type PostResult struct {
	Err error
	URL string
}

// RemotePost is a normalized remote platform post used by keyword watch.
type RemotePost struct {
	RemoteID     string
	RemoteURL    string
	AuthorID     string
	AuthorHandle string
	Content      string
	PostedAt     time.Time
}

// MatchSink receives posts that matched one or more keywords.
type MatchSink func(post *RemotePost, matchedKeywords []string)

// WatchHandle stops a native keyword watch session.
type WatchHandle interface {
	Stop()
}

// NativeFidelity describes how closely native watch matches exact keywords.
const (
	NativeFidelityExactRule    = "exact_rule"
	NativeFidelityClientFilter = "client_filter"
	NativeFidelityHashtagOnly  = "hashtag_only"
	NativeFidelitySearchOnly   = "search_only"
)

// ConnectorWithKeywordWatch can run a push/stream subscription for keywords.
type ConnectorWithKeywordWatch interface {
	PreferNativeWatch(sa *SocialAccount, keywords []string) bool
	NativeWatchFidelity(sa *SocialAccount, keywords []string) string
	StartKeywordWatch(ctx context.Context, sa *SocialAccount, keywords []string, sink MatchSink) (WatchHandle, error)
}

// ConnectorWithKeywordSearch can poll recent posts by keyword.
type ConnectorWithKeywordSearch interface {
	SearchRecentByKeyword(ctx context.Context, sa *SocialAccount, keyword string, since time.Time, limit int) ([]*RemotePost, error)
}

// RateLimitedError signals that the platform asked the client to back off.
type RateLimitedError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "rate limited"
}

// AsRateLimited returns *RateLimitedError if err (or a wrapped err) is one.
func AsRateLimited(err error) *RateLimitedError {
	if err == nil {
		return nil
	}
	if rl, ok := err.(*RateLimitedError); ok {
		return rl
	}
	return nil
}
