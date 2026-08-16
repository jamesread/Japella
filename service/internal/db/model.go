package db

import (
	"database/sql"
	"time"
)

type Model struct {
	/**
	We use uint32 for IDs which might seem a bit unusual in 2025, but JavaScript
	uses 53-bit integers, and so all ints have to wrapped to a string, which
	gets way too ugly.

	A uint32 can hold 4,294,967,295 unique values, which "should be enough for anybody".

	I'm looking forward to the bug report when someone eventually does go over
	4 billion rows, but maybe JavaScript will have a better way of handling 64-bit
	integers by then.
	*/
	ID        uint32    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type SocialAccount struct {
	Model

	Connector          string         `db:"connector"`
	Identity           string         `db:"identity"`
	Did                string         `db:"did"`
	Homeserver         string         `db:"homeserver"`
	OAuth2Token        string         `db:"oauth2_token"`
	OAuth2TokenExpiry  time.Time      `db:"oauth2_token_expiry"`
	OAuth2RefreshToken string         `db:"oauth2_refresh_token"`
	DpopKey            sql.NullString `db:"dpop_key"`
	Active             bool           `db:"active"`
	State              string         `db:"state"`
	OwnerUserID        sql.NullInt32  `db:"owner_user_id"`
}

type SocialAccountShare struct {
	ID              uint32    `db:"id"`
	SocialAccountID uint32    `db:"social_account_id"`
	UserGroupID     uint32    `db:"user_group_id"`
	CanRead         bool      `db:"can_read"`
	CanPost         bool      `db:"can_post"`
	CanManage       bool      `db:"can_manage"`
	CreatedAt       time.Time `db:"created_at"`
	GroupName       string    `db:"group_name"`
}

type CannedPost struct {
	Model

	Content string `db:"content"`
}

type Post struct {
	Model

	SocialAccountID    sql.NullInt32 `db:"social_account_id"`
	SocialAccount      *SocialAccount
	Status             bool           `db:"status"`
	State              string         `db:"state"`
	Content            string         `db:"content"`
	PostURL            sql.NullString `db:"post_url"`
	RemoteID           sql.NullString `db:"remote_id"`
	ScheduledAt        sql.NullTime   `db:"scheduled_at"`
	CampaignID         sql.NullInt32  `db:"campaign_id"`
	CampaignName       sql.NullString `db:"campaign_name"`
	SubmissionSource   string         `db:"submission_source"`
	SubmittedByUserID  sql.NullInt32  `db:"submitted_by_user_id"`
	AccountPolicyID    sql.NullInt32  `db:"account_policy_id"`
	ApprovalStage      uint32         `db:"approval_stage"`
}

func NullSocialAccountID(id uint32) sql.NullInt32 {
	if id == 0 {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(id), Valid: true}
}

func (p *Post) SocialAccountIDUint() uint32 {
	if p == nil || !p.SocialAccountID.Valid {
		return 0
	}
	return uint32(p.SocialAccountID.Int32)
}

func (p *Post) SocialAccountIDPtr() *uint32 {
	if p == nil || !p.SocialAccountID.Valid {
		return nil
	}
	id := p.SocialAccountIDUint()
	return &id
}

// AccountPolicy is a named approval configuration attachable to social accounts.
type AccountPolicy struct {
	Model

	Name        string `db:"name"`
	Description string `db:"description"`
	ApplyToMCP  bool   `db:"apply_to_mcp"`
	ApplyToUI   bool   `db:"apply_to_ui"`
}

// AccountPolicyApprovalStage is one ordered approval step (user XOR usergroup).
type AccountPolicyApprovalStage struct {
	Model

	AccountPolicyID uint32         `db:"account_policy_id"`
	StageOrder      uint32         `db:"stage_order"`
	UserID          sql.NullInt32  `db:"user_id"`
	UserGroupID     sql.NullInt32  `db:"user_group_id"`
	Username        sql.NullString `db:"username"`
	UserGroupName   sql.NullString `db:"user_group_name"`
}

// PostApproval records that a stage was approved for a post.
type PostApproval struct {
	Model

	PostID           uint32 `db:"post_id"`
	StageID          uint32 `db:"stage_id"`
	ApprovedByUserID uint32 `db:"approved_by_user_id"`
}

// How a user_accounts row was provisioned (see CreateUserAccount).
const (
	UserCreatedByAdmin = "admin-created"
	UserCreatedBySSO   = "sso-autocreated"
)

type UserAccount struct {
	Model

	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	CreatedBy    string `db:"created_by"`
}

type UserGroup struct {
	Model

	Name string `db:"name"`
}

type UserGroupMembership struct {
	Model

	UserAccountID uint32 `db:"user_account_id"`
	UserAccount   *UserAccount

	UserGroupID uint32 `db:"user_group_id"`
	UserGroup   *UserGroup
}

type ApiKey struct {
	Model

	Name          string       `db:"name"`
	KeyValue      string       `db:"key_value"` // Key keyword in SQL
	LastUsedAt    sql.NullTime `db:"last_used_at"`
	UserAccountID uint32       `db:"user_account_id"`
	UserAccount   *UserAccount
}

type Session struct {
	Model

	UserAccountID      uint32        `db:"user_account_id"`
	UserAccount        *UserAccount
	SID                string        `db:"sid"` // Session ID
	ImpersonatorUserID sql.NullInt32 `db:"impersonator_user_id"`
}

type Cvar struct {
	Model

	KeyName      string `db:"key_name"`
	Title        string `db:"title"`
	ValueString  string `db:"value_string"`
	ValueInt     int32  `db:"value_int"`
	Description  string `db:"description"`
	DefaultValue string `db:"default_value"`
	Category     string `db:"category"`
	Type         string `db:"type"`
	DocsUrl      string `db:"docs_url"`
	ExternalUrl  string `db:"external_url"` // URL to the external documentation or portal
}

type UserPreferences struct {
	Model

	UserAccountID  uint32 `db:"user_account_id"`
	UserAccount    UserAccount
	Language       string `db:"language"`
	SidebarEnabled bool   `db:"sidebar_enabled"`
	ThemeToggleEnabled bool `db:"theme_toggle_enabled"`
}

type Campaign struct {
	Model

	Name         string     `db:"name"`
	Description  string     `db:"description"`
	PostCount    int32      `db:"post_count"`
	LastPostDate *time.Time `db:"last_post_date"`
	StartDate    time.Time  `db:"start_date"`
	EndDate      time.Time  `db:"end_date"`
	AccountCount int32      `db:"account_count"`
}

type Feed struct {
	Model

	SocialAccountID        uint32    `db:"social_account_id"`
	Content                string    `db:"content"`
	PostedDate             time.Time `db:"posted_date"`
	AuthorID               string    `db:"author_id"`
	AuthorName             string    `db:"author_name"`
	AuthorAvatarURL        string    `db:"author_avatar_url"`
	RemoteURL              string    `db:"remote_url"`
	RemoteID               string    `db:"remote_id"`
	PreviewURL             string    `db:"preview_url"`
	PreviewTitle           string    `db:"preview_title"`
	PreviewDescription     string    `db:"preview_description"`
	PreviewImageURL        string    `db:"preview_image_url"`
	SocialAccountIdentity  string    `db:"social_account_identity"`
	SocialAccountConnector string    `db:"social_account_connector"`
}

type TableLog struct {
	Model

	Message                       string         `db:"message"`
	Level                         string         `db:"level"`
	RelatedSocialAccountID        sql.NullInt32  `db:"related_social_account_id"`
	RelatedSocialAccountIdentity  sql.NullString `db:"related_social_account_identity"`
	RelatedSocialAccountConnector sql.NullString `db:"related_social_account_connector"`
}

type WebhookHook struct {
	Model

	Connector string `db:"connector"`
	Identity  string `db:"identity"`
	BotID     string `db:"bot_id"`
	URL       string `db:"url"`
	Enabled   bool   `db:"enabled"`
}

type ChatBotInstance struct {
	Model

	Protocol    string `db:"protocol"`
	BotID       string `db:"bot_id"`
	DisplayName string `db:"display_name"`
}

type ChatBotMessage struct {
	Model

	Connector         string `db:"connector"`
	Identity          string `db:"identity"`
	BotID             string `db:"bot_id"`
	ConversationKey   string `db:"conversation_key"`
	ConversationTitle string `db:"conversation_title"`
	Channel           string `db:"channel"`
	Author            string `db:"author"`
	Content           string `db:"content"`
	Direction         string `db:"direction"` // incoming|outgoing
	MessageID         string `db:"message_id"`
	TimestampUnix     int64  `db:"timestamp_unix"`
}
