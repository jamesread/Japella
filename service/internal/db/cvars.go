package db

var CvarKeys = struct {
	OAuth2RedirectURL string

	XClientID     string
	XClientSecret string
}{
	OAuth2RedirectURL: "oauth2_redirect_url",

	XClientID:     "twitter_client_id",
	XClientSecret: "twitter_client_secret",
}

var CvarList = []Cvar{
	{KeyName: CvarKeys.OAuth2RedirectURL, Title: "OAuth2 Redirect URL", ValueString: "http://localhost:8080/oauth2callback", Category: "OAuth2", Description: "The redirect URL for OAuth2 authentication", Type: "text"},

	{KeyName: CvarKeys.XClientID, Title: "X Client ID", ValueString: "", Category: "X", Description: "Client ID for X OAuth", Type: "text"},
	{KeyName: CvarKeys.XClientSecret, Title: "X Client Secret", ValueString: "", Category: "X", Description: "Client secret for X OAuth", Type: "password"},
}

func (db *DB) InsertCvarsIfNotExists() error {
	for _, cvar := range CvarList {
		err := db.InsertCvarIfNotExists(&cvar)

		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) InsertCvarIfNotExists(cvar *Cvar) error {
	db.Logger().Infof("Inserting cvar: %s", cvar.KeyName)

	_, err := db.ResilientExec(`INSERT IGNORE INTO cvars (key_name, title, value_string, value_int, description, default_value, category, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		cvar.KeyName, cvar.Title, cvar.ValueString, cvar.ValueInt, cvar.Description, cvar.DefaultValue, cvar.Category, cvar.Type)

	return err
}
