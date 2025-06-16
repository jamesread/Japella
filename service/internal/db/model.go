package db;

import (
	"gorm.io/gorm"
)

type SocialAccount struct {
	gorm.Model

	ID         uint32 `gorm:"primarykey"`
	Connector  string
	Identity   string
	OAuthToken string
	Active bool
}

type CannedPost struct {
	gorm.Model

	ID      uint32 `gorm:"primarykey"`
	Content string
}

type Post struct {
	gorm.Model

	ID uint32 `gorm:"primarykey"`
	SocialAccountID uint32
	SocialAccount SocialAccount `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Status bool
	Content string
	PostURL string
	RemoteID string
}
