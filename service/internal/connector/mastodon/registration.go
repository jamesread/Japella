package mastodon

import (
	"github.com/jamesread/japella/internal/db"
)

const defaultMastodonInstance = "https://mastodon.social"

func (c *MastodonConnector) defaultInstanceURL() string {
	return defaultMastodonInstance
}

func (c *MastodonConnector) IsRegistered() bool {
	clientID := c.db.GetCvarString(CFG_MASTODON_CLIENT_ID)
	clientSecret := c.db.GetCvarString(CFG_MASTODON_CLIENT_SECRET)

	return clientID != "" && clientSecret != ""
}

func (c *MastodonConnector) tryRegisterClientIfNeeded() {
	if c.db == nil || c.IsRegistered() {
		return
	}

	redirectURL := c.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL)
	if redirectURL == "" {
		c.Logger().Warnf("Mastodon OAuth app not registered: OAuth2 redirect URL is not configured")
		return
	}

	if err := c.RegisterClient(); err != nil {
		c.Logger().Warnf("Mastodon OAuth auto-registration failed: %v", err)
		return
	}

	c.Logger().Infof("Mastodon OAuth app auto-registered with redirect URI %s", redirectURL)
}
