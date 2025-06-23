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
	ID        uint32 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SocialAccount struct {
	Model

	Connector          string `gorm:"type:varchar(50);not null"`
	Identity           string `gorm:"type:varchar(255);not null"`
	OAuth2Token        string `gorm:"type:text"`
	OAuth2TokenExpiry  time.Time
	OAuth2RefreshToken string `gorm:"type:text"`
	Active             bool   `gorm:"default:false;not null"`
}

type CannedPost struct {
	Model

	Content string `gorm:"type:text;not null"`
}

type Post struct {
	Model

	SocialAccountID uint32         `gorm:"not null"`
	SocialAccount   *SocialAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	Status          bool           `gorm:"default:false;not null"`
	Content         string         `gorm:"type:text;not null"`
	PostURL         string         `gorm:"type:varchar(500)"`
	RemoteID        string         `gorm:"type:varchar(255)"`
}

type UserAccount struct {
	Model

	Username     string `gorm:"type:varchar(64);uniqueIndex;not null"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
}

type UserGroup struct {
	Model

	Name string `gorm:"type:varchar(100);not null"`
}

type UserGroupMembership struct {
	Model

	UserAccountID uint32       `gorm:"uniqueIndex:idx_user_group_membership;not null"`
	UserAccount   *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`

	UserGroupID uint32     `gorm:"uniqueIndex:idx_user_group_membership;not null"`
	UserGroup   *UserGroup `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
}

type ApiKey struct {
	Model

	KeyValue      string       `gorm:"type:varchar(64);uniqueIndex;not null"` // Key keyword in SQL
	UserAccountID uint32       `gorm:"not null"`
	UserAccount   *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
}

type Session struct {
	Model

	UserAccountID uint32       `gorm:"not null"`
	UserAccount   *UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	SID           string       `gorm:"uniqueIndex;type:varchar(255);not null"` // Session ID
}

type Cvar struct {
	Model

	KeyName      string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Title        string `gorm:"type:varchar(100);not null"`
	ValueString  string `gorm:"type:text"`
	ValueInt     int32  `gorm:"type:int;default:0"`
	Description  string `gorm:"type:text"`
	DefaultValue string `gorm:"type:text"`
	Category     string `gorm:"type:varchar(50)"`
	Type         string `gorm:"type:varchar(20);not null"`
}

type UserPreferences struct {
	Model

	UserAccountID uint32      `gorm:"uniqueIndex;not null"`
	UserAccount   UserAccount `gorm:"constraint:OnUpdate:NO ACTION,OnDelete:NO ACTION;"`
	Language      string      `gorm:"type:varchar(10);default:'en'"`
}
