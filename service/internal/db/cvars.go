package db;

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var CvarKeys = struct {
	OAuth2RedirectURL string

	MastodonClientID     string
	MastodonClientSecret string

	XClientID      string
	XClientSecret   string
}{
	OAuth2RedirectURL: "oauth2_redirect_url",

	MastodonClientID:     "mastodon_client_id",
	MastodonClientSecret: "mastodon_client_secret",

	XClientID:      "twitter_client_id",
	XClientSecret:  "twitter_client_secret",
}

var CvarList = []Cvar{
	{KeyName: CvarKeys.OAuth2RedirectURL, Title: "OAuth2 Redirect URL", ValueString: "http://localhost:8080/oauth2callback", Category: "OAuth2", Description: "The redirect URL for OAuth2 authentication", Type: "text"},

	{KeyName: CvarKeys.MastodonClientID, Title: "Mastodon Client ID", ValueString: "", Category: "Mastodon", Description: "Client ID for Mastodon OAuth", Type: "text"},
	{KeyName: CvarKeys.MastodonClientSecret, Title: "Mastodon Client Secret", ValueString: "", Category: "Mastodon", Description: "Client secret for Mastodon OAuth", Type: "password"},

	{KeyName: CvarKeys.XClientID, Title: "X Client ID", ValueString: "", Category: "X", Description: "Client ID for X OAuth", Type: "text"},
	{KeyName: CvarKeys.XClientSecret, Title: "X Client Secret", ValueString: "", Category: "X", Description: "Client secret for X OAuth", Type: "password"},
}

func (db *DB) InsertCvarsIfNotExists() {
	for _, cvar := range CvarList {
		var existing Cvar
		result := db.conn.Where("key_name = ?", cvar.KeyName).First(&existing)

		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			log.Errorf("Error checking for existing cvar: %v", result.Error)
			continue
		}

		if result.RowsAffected == 0 {
			log.Infof("Inserting new cvar: %s", cvar.KeyName)
			db.conn.Create(&cvar)
		} else {
			log.Infof("Cvar already exists: %s", cvar.KeyName)
		}
	}
}
