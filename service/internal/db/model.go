package db;

import (
	"gorm.io/gorm"
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
	ID        uint32 `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type SocialAccount struct {
	Model

	Connector  string
	Identity   string
	OAuthToken string
	Active bool
}

type CannedPost struct {
	Model

	ID      uint32 `gorm:"primarykey"`
	Content string
}

type Post struct {
	Model

	ID uint32 `gorm:"primarykey"`
	SocialAccountID uint32
	SocialAccount SocialAccount `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Status bool
	Content string
	PostURL string
	RemoteID string
}

type UserAccount struct {
	Model

	ID       uint32 `gorm:"primarykey"`
	Username string `gorm:"uniqueIndex"`

}

type UserGroup struct {
	Model

	ID          uint32 `gorm:"primarykey"`
	Name        string `gorm:"uniqueIndex"`
}

type UserGroupMembership struct {
	Model
}

type ApiKey struct {
	Model

	ID uint32 `gorm:"primarykey"`
	Key string `gorm:"uniqueIndex"`
	UserAccountID uint32
	UserAccount UserAccount `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
