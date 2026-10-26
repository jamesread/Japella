package db

import "github.com/jamesread/japella/internal/debuglog"

const (
	ThemeControlSystem = "system"
	ThemeControlUser   = "user"
)

var CvarKeys = struct {
	BaseUrl                           string
	OAuth2RedirectURL                 string
	IsPubliclyAccessible              string
	ThemeColorSchemeSwitcherEnabled   string
	ThemeName                         string
	ThemeControl                      string
	DebugAuth                         string
	DebugFeed                         string
	DebugHTTP                         string
	KeywordWatchSearchIntervalSeconds string
	KeywordWatchSearchLookbackSeconds string
	KeywordWatchNativeRetrySeconds    string
	KeywordWatchMaxMatchesPerPoll     string
	KeywordWatchMaxEmitsPerHour       string
	KeywordWatchMatchRetention        string
	PostEngagementIntervalSeconds     string
	PostEngagementMaxAgeDays          string
	PostEngagementMaxPostsPerRun      string
}{
	BaseUrl:                           "base_url",
	OAuth2RedirectURL:                 "oauth2_redirect_url",
	IsPubliclyAccessible:              "is_publicly_accessible",
	ThemeColorSchemeSwitcherEnabled:   "theme_color_scheme_switcher_enabled",
	ThemeName:                         "theme_name",
	ThemeControl:                      "theme_control",
	DebugAuth:                         debuglog.KeyAuth,
	DebugFeed:                         debuglog.KeyFeed,
	DebugHTTP:                         debuglog.KeyHTTP,
	KeywordWatchSearchIntervalSeconds: "keyword_watch_search_interval_seconds",
	KeywordWatchSearchLookbackSeconds: "keyword_watch_search_lookback_seconds",
	KeywordWatchNativeRetrySeconds:    "keyword_watch_native_retry_seconds",
	KeywordWatchMaxMatchesPerPoll:     "keyword_watch_max_matches_per_poll",
	KeywordWatchMaxEmitsPerHour:       "keyword_watch_max_emits_per_hour",
	KeywordWatchMatchRetention:        "keyword_watch_match_retention",
	PostEngagementIntervalSeconds:     "post_engagement_interval_seconds",
	PostEngagementMaxAgeDays:          "post_engagement_max_age_days",
	PostEngagementMaxPostsPerRun:      "post_engagement_max_posts_per_run",
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
	{KeyName: CvarKeys.KeywordWatchSearchIntervalSeconds, Title: "Keyword watch search interval (seconds)", ValueInt: 300, Category: "Keyword Watch", Description: "How often Japella polls platform search APIs when native watch is unavailable or degraded.", Type: "int", DefaultValue: "300"},
	{KeyName: CvarKeys.KeywordWatchSearchLookbackSeconds, Title: "Keyword watch search lookback (seconds)", ValueInt: 900, Category: "Keyword Watch", Description: "How far back each search poll considers recent posts.", Type: "int", DefaultValue: "900"},
	{KeyName: CvarKeys.KeywordWatchNativeRetrySeconds, Title: "Keyword watch native retry (seconds)", ValueInt: 1800, Category: "Keyword Watch", Description: "Minimum wait before re-probing native watch after falling back to search.", Type: "int", DefaultValue: "1800"},
	{KeyName: CvarKeys.KeywordWatchMaxMatchesPerPoll, Title: "Keyword watch max matches per poll", ValueInt: 50, Category: "Keyword Watch", Description: "Circuit breaker: max matches processed per account per search poll.", Type: "int", DefaultValue: "50"},
	{KeyName: CvarKeys.KeywordWatchMaxEmitsPerHour, Title: "Keyword watch max webhook emits per hour", ValueInt: 100, Category: "Keyword Watch", Description: "Per social account cap on keyword.matched webhook emissions per hour.", Type: "int", DefaultValue: "100"},
	{KeyName: CvarKeys.KeywordWatchMatchRetention, Title: "Keyword watch match retention", ValueInt: 200, Category: "Keyword Watch", Description: "How many recent matches to keep per keyword watch.", Type: "int", DefaultValue: "200"},
	{KeyName: CvarKeys.PostEngagementIntervalSeconds, Title: "Post engagement fetch interval (seconds)", ValueInt: 900, Category: "Post Engagement", Description: "How often Japella fetches likes, reposts and replies for recently published posts. Each post costs one API call per run on the platform.", Type: "int", DefaultValue: "900"},
	{KeyName: CvarKeys.PostEngagementMaxAgeDays, Title: "Post engagement max post age (days)", ValueInt: 7, Category: "Post Engagement", Description: "Only posts published within this many days are refreshed. Set to 0 to disable engagement fetching.", Type: "int", DefaultValue: "7"},
	{KeyName: CvarKeys.PostEngagementMaxPostsPerRun, Title: "Post engagement max posts per run", ValueInt: 100, Category: "Post Engagement", Description: "Upper bound on posts refreshed per run; the least recently refreshed posts go first.", Type: "int", DefaultValue: "100"},
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
