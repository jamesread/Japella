package db

var CvarKeys = struct {
	BaseUrl string
	OAuth2RedirectURL string
}{
	BaseUrl:          "base_url",
	OAuth2RedirectURL: "oauth2_redirect_url",
}

var CvarList = []Cvar{
	{KeyName: CvarKeys.BaseUrl, Title: "Base URL", ValueString: "http://localhost:8080", Category: "General", Description: "The base URL of the application", Type: "text"},
	{KeyName: CvarKeys.OAuth2RedirectURL, Title: "OAuth2 Redirect URL", ValueString: "http://localhost:8080/oauth2callback", Category: "OAuth2", Description: "The redirect URL for OAuth2 authentication", Type: "text"},
}

func (db *DB) InsertCvarsIfNotExists(chain *ConnectionChain) {
	for _, cvar := range CvarList {
		err := db.InsertCvarIfNotExists(&cvar)

		if err != nil {
			chain.err = err
			return
		}
	}
}

func (db *DB) InsertCvarIfNotExists(cvar *Cvar) error {
	res, err := db.ResilientExec(`INSERT INTO cvars (key_name, title, value_string, value_int, description, default_value, category, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE description = ?, docs_url = ?, external_url = ?`,
		cvar.KeyName, cvar.Title, cvar.ValueString, cvar.ValueInt, cvar.Description, cvar.DefaultValue, cvar.Category, cvar.Type, cvar.Description, cvar.DocsUrl, cvar.ExternalUrl)

	if err != nil {
		db.Logger().Errorf("Failed to insert cvar %s: %v", cvar.KeyName, err)
		return err
	}

	count, err := res.RowsAffected()

	if err != nil {
		db.Logger().Errorf("Failed to get affected rows for cvar %s: %v", cvar.KeyName, err)
		return err
	}

	if count > 0 {
		db.Logger().Infof("Cvar %s inserted successfully", cvar.KeyName)
	}

	return err
}
