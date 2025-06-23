package db;

import (
	"time"
)

type Model struct {
	/**
	We use uint32 for IDs which might seem a but unusual in 2025, but JavaScript
	uses 53-bit integers, and so all ints have to wrapped to a string, which
	gets way too ugly.

	A uint32 can hold 4,294,967,295 unique values, which "should be enough for anybody".

	I'm looking forward to the bug report when someone eventually does go over
	4 billion rows, but maybe JavaScript will have a better way of handling 64-bit
	integers by then.
	*/
	ID        uint32 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SocialAccount struct {
	Model

	Connector  string
	Identity   string
	OAuth2Token string
	OAuth2TokenExpiry time.Time
	OAuth2RefreshToken string
	Active bool
}

type CannedPost struct {
	Model

	Content string
}

type Post struct {
	Model

	SocialAccountID uint32
	SocialAccount *SocialAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	Status bool
	Content string
	PostURL string
	RemoteID string
}

type UserAccount struct {
	Model

	Username string `gorm:"type:varchar(64);uniqueIndex"`
	PasswordHash string
}

type UserGroup struct {
	Model

	Name        string
}

type UserGroupMembership struct {
	Model

	UserAccountID uint32 `gorm:"uniqueIndex:idx_user_group_membership"`
	UserAccount *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`

	UserGroupID   uint32 `gorm:"uniqueIndex:idx_user_group_membership"`
	UserGroup *UserGroup `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
}

type ApiKey struct {
	Model

	KeyValue string `gorm:"type:varchar(64);uniqueIndex"` // Key keyword in SQL
	UserAccountID uint32
	UserAccount *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
}

type Session struct {
	Model

	UserAccountID uint32
	UserAccount *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	SID string `gorm:"uniqueIndex"` // Session ID
}

type Cvar struct {
	Model

	KeyName  string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Title	   string `gorm:"type:varchar(100);not null"`
	ValueString    string `gorm:"type:text"`
	ValueInt int32 `gorm:"type:int"`
	Description string `gorm:"type:text"`
	DefaultValue string `gorm:"type:text"`
	Category string `gorm:"type:text"`
	Type string `gorm:"type:varchar(20);not null"`
}

type UserPreferences struct {
	Model

	UserAccountID uint32 `gorm:"uniqueIndex"`
	UserAccount UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	Language string `gorm:"type:varchar(10)"`
}
