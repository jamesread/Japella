package db

import "github.com/jamesread/japella/internal/debuglog"

const (
	ThemeControlSystem = "system"
	ThemeControlUser   = "user"
)

var CvarKeys = struct {
	BaseUrl                        string
	OAuth2RedirectURL              string
	IsPubliclyAccessible           string
	ThemeColorSchemeSwitcherEnabled string
	ThemeName                      string
	ThemeControl                   string
	DebugAuth                      string
	DebugFeed                      string
	DebugHTTP                      string
}{
	BaseUrl:                         "base_url",
	OAuth2RedirectURL:               "oauth2_redirect_url",
	IsPubliclyAccessible:            "is_publicly_accessible",
	ThemeColorSchemeSwitcherEnabled: "theme_color_scheme_switcher_enabled",
	ThemeName:                       "theme_name",
	ThemeControl:                    "theme_control",
	DebugAuth:                       debuglog.KeyAuth,
	DebugFeed:                       debuglog.KeyFeed,
	DebugHTTP:                       debuglog.KeyHTTP,
}

var CvarList = []Cvar{
	{KeyName: CvarKeys.BaseUrl, Title: "Base URL", ValueString: "http://localhost:8080", Category: "General", Description: "The base URL of the application", Type: "text"},
	{KeyName: CvarKeys.IsPubliclyAccessible, Title: "Is Publicly Accessible", ValueInt: 0, Category: "General", Description: "When enabled, OAuth connectors (Mastodon, X, Bluesky, Facebook, Instagram) are started. When disabled, they appear in Unregistered Connectors.", Type: "bool", DefaultValue: "0"},
	{KeyName: CvarKeys.OAuth2RedirectURL, Title: "OAuth2 Redirect URL", ValueString: "http://localhost:8080/oauth2callback", Category: "OAuth2", Description: "The redirect URL for OAuth2 authentication", Type: "text"},
	{KeyName: CvarKeys.ThemeColorSchemeSwitcherEnabled, Title: "Color scheme switcher", ValueInt: 0, Category: "Theme", Description: "Show the auto/light/dark color scheme control in the PicoCrank header.", Type: "bool", DefaultValue: "0"},
	{KeyName: CvarKeys.ThemeName, Title: "Theme name", ValueString: "", Category: "Theme", Description: "Default or enforced drop-in CSS theme (empty = Femtocrank base styling only).", Type: "text", DefaultValue: ""},
	{KeyName: CvarKeys.ThemeControl, Title: "Theme control", ValueString: ThemeControlUser, Category: "Theme", Description: "System preference forces the theme name cvar for everyone. User preference uses the cvar as default and allows overrides on User Preferences.", Type: "text", DefaultValue: ThemeControlUser},
	{KeyName: CvarKeys.DebugAuth, Title: "Auth debug logging", ValueInt: 0, Category: "Debug", Description: "Log authentication details (API key checks, httpauthshim path). Independent of config.yaml logLevel.", Type: "bool", DefaultValue: "0"},
	{KeyName: CvarKeys.DebugFeed, Title: "Feed fetcher debug logging", ValueInt: 0, Category: "Debug", Description: "Log feed fetch cycles and per-post insert/skip details. Independent of config.yaml logLevel.", Type: "bool", DefaultValue: "0"},
	{KeyName: CvarKeys.DebugHTTP, Title: "HTTP client debug logging", ValueInt: 0, Category: "Debug", Description: "Log outbound HTTP requests, responses, headers, and bodies. Independent of config.yaml logLevel.", Type: "bool", DefaultValue: "0"},
}

func (db *DB) ThemeColorSchemeSwitcherEnabled() bool {
	if db.GetCvar(CvarKeys.ThemeColorSchemeSwitcherEnabled) != nil {
		return db.GetCvarBool(CvarKeys.ThemeColorSchemeSwitcherEnabled)
	}
	return db.GetCvarBool("enable_theme_color_scheme_switcher")
}

func (db *DB) ThemeName() string {
	return db.GetCvarString(CvarKeys.ThemeName)
}

func (db *DB) ThemeControl() string {
	value := db.GetCvarString(CvarKeys.ThemeControl)
	if value == ThemeControlSystem {
		return ThemeControlSystem
	}
	return ThemeControlUser
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

// LoadDebugLogFlags reads debug.* cvars into the in-memory debuglog registry.
func (db *DB) LoadDebugLogFlags(chain *ConnectionChain) {
	flags := map[string]bool{
		CvarKeys.DebugAuth: db.GetCvarBool(CvarKeys.DebugAuth),
		CvarKeys.DebugFeed: db.GetCvarBool(CvarKeys.DebugFeed),
		CvarKeys.DebugHTTP: db.GetCvarBool(CvarKeys.DebugHTTP),
	}
	debuglog.Init(flags)
	db.Logger().Infof("Loaded debug log flags: auth=%v feed=%v http=%v",
		flags[CvarKeys.DebugAuth], flags[CvarKeys.DebugFeed], flags[CvarKeys.DebugHTTP])
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
