package db

import (
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

	Connector          string    `db:"connector"`
	Identity           string    `db:"identity"`
	OAuth2Token        string    `db:"oauth2_token"`
	OAuth2TokenExpiry  time.Time `db:"oauth2_token_expiry"`
	OAuth2RefreshToken string    `db:"oauth2_refresh_token"`
	Active             bool      `db:"active"`
}

type CannedPost struct {
	Model

	Content string `db:"content"`
}

type Post struct {
	Model

	SocialAccountID uint32 `db:"social_account_id"`
	SocialAccount   *SocialAccount
	Status          bool   `db:"status"`
	Content         string `db:"content"`
	PostURL         string `db:"post_url"`
	RemoteID        string `db:"remote_id"`
}

type UserAccount struct {
	Model

	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
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

	KeyValue      string `db:"key_value"` // Key keyword in SQL
	UserAccountID uint32 `db:"user_account_id"`
	UserAccount   *UserAccount
}

type Session struct {
	Model

	UserAccountID uint32 `db:"user_account_id"`
	UserAccount   *UserAccount
	SID           string `db:"sid"` // Session ID
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

	UserAccountID uint32 `db:"user_account_id"`
	UserAccount   UserAccount
	Language      string `db:"language"`
}
