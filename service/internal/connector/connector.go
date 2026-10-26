package connector

import (
	"context"
	"net/http"
	"strconv"
	"strings"
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

// PostEngagement is a point-in-time snapshot of public engagement counters
// for a post Japella published. Platforms that lack a counter leave it at 0.
type PostEngagement struct {
	Likes       uint32
	Reposts     uint32
	Replies     uint32
	Quotes      uint32
	Impressions uint32
}

// ConnectorWithPostEngagement can fetch engagement counters (likes, reposts,
// replies, ...) for a post previously published through PostToWall.
// postURL is the URL returned in PostResult; remoteID may be empty.
type ConnectorWithPostEngagement interface {
	FetchPostEngagement(sa *SocialAccount, remoteID string, postURL string) (*PostEngagement, error)
}

// PostReply is one reply in the conversation under a post Japella published.
// ParentRemoteID is the remote id of the post it replies to: either the
// published post itself (RootRemoteID) or another reply.
type PostReply struct {
	RemoteID        string
	ParentRemoteID  string
	RemoteURL       string
	AuthorName      string
	AuthorHandle    string
	AuthorAvatarURL string
	// Content is plain text; connectors strip any platform HTML.
	Content    string
	PostedDate time.Time
}

// PostReplies is the reply tree under a published post, flattened.
type PostReplies struct {
	// RootRemoteID is the platform id of the published post, as replies
	// reference it in ParentRemoteID.
	RootRemoteID string
	Replies      []*PostReply
}

// ConnectorWithPostReplying can publish a reply to a post as the account.
type ConnectorWithPostReplying interface {
	ReplyToPost(sa *SocialAccount, remoteID string, remoteURL string, content string) (*PostReply, error)
}

// ConnectorWithPostReplies can fetch the replies (including nested replies)
// to a post previously published through PostToWall.
type ConnectorWithPostReplies interface {
	FetchPostReplies(sa *SocialAccount, remoteID string, postURL string) (*PostReplies, error)
}

// ChatBot represents a chatbot instance from a connector
type ChatBot struct {
	BotID               string // Immutable instance id (e.g. botA)
	Connector           string // The connector protocol (e.g., "telegram", "discord")
	Name                string // Display name of the bot
	Identity            string // Bot identity/username on the platform (after connect)
	Icon                string // Icon identifier
	IsRunning           bool   // Whether the bot is currently running
	StatusMessage       string // Diagnostic message explaining why the bot is stopped or its current status
	ErrorMessage        string // Error message if the bot failed to start
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
	// Handles are the full platform handle (e.g. "@alice@example.social");
	// profile URLs open the author's page on their platform.
	AuthorHandle             string
	AuthorProfileURL         string
	OriginalAuthorHandle     string
	OriginalAuthorProfileURL string
	RemoteURL                string
	RemoteID                 string
	PreviewURL               string
	PreviewTitle             string
	PreviewDescription       string
	PreviewImageURL          string
	// Language is the platform's language tag (e.g. "en"), if it reports one.
	Language string

	PostInteraction
}

// PostInteraction is what's needed to like or open the thread of a post,
// and its state as the fetching account sees it.
type PostInteraction struct {
	// InteractionRemoteID is the id to like or fetch the thread of, when it
	// differs from RemoteID (a boost's original status).
	InteractionRemoteID string
	Liked               bool
	LikeCount           uint32
	ReplyCount          uint32
}

// LikeResult is a post's like state after liking or unliking it.
type LikeResult struct {
	Liked     bool
	LikeCount uint32
}

// ConnectorWithPostLikes can like and unlike posts as the social account.
type ConnectorWithPostLikes interface {
	SetPostLiked(sa *SocialAccount, remoteID string, remoteURL string, liked bool) (*LikeResult, error)
}

// FollowState is whether the social account follows an author.
type FollowState struct {
	Following bool
	// Requested is set while a follow awaits the author's approval.
	Requested bool
}

// ConnectorWithAuthorFollow can follow and unfollow authors as the social
// account. authorID is the author id the connector put on feed posts.
type ConnectorWithAuthorFollow interface {
	GetFollowState(sa *SocialAccount, authorID string) (*FollowState, error)
	SetFollowing(sa *SocialAccount, authorID string, follow bool) (*FollowState, error)
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

	// Optional display details, used when showing posts in feed columns.
	AuthorName       string
	AuthorAvatarURL  string
	AuthorProfileURL string

	// ContentIsHTML is true when Content is platform HTML rather than plain text.
	ContentIsHTML bool

	// Language is the platform's language tag (e.g. "en"), if it reports one.
	Language string

	PostInteraction
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

// ConnectorWithHashtagTimeline can list recent posts carrying a hashtag
// directly, which some platforms do more reliably than keyword search.
// tag has no leading '#'.
type ConnectorWithHashtagTimeline interface {
	FetchHashtagTimeline(ctx context.Context, sa *SocialAccount, tag string, limit int) ([]*RemotePost, error)
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

// RateLimitedFromResponse returns a *RateLimitedError when res is an HTTP 429,
// honouring a numeric Retry-After header when present. It returns nil otherwise.
func RateLimitedFromResponse(res *http.Response) *RateLimitedError {
	if res == nil || res.StatusCode != http.StatusTooManyRequests {
		return nil
	}
	retryAfter := time.Minute
	if secs, err := strconv.Atoi(strings.TrimSpace(res.Header.Get("Retry-After"))); err == nil && secs > 0 {
		retryAfter = time.Duration(secs) * time.Second
	}
	return &RateLimitedError{RetryAfter: retryAfter, Message: "rate limited by platform"}
}
