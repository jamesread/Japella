package instagram

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type InstagramConnector struct{}

func (c *InstagramConnector) SetStartupConfiguration(*connector.ControllerStartupConfiguration) {}

func (c *InstagramConnector) Start() {
	log.Infof("Instagram connector started (stub implementation)")
}

func (c *InstagramConnector) GetIdentity() string {
	return "Instagram"
}

func (c *InstagramConnector) GetProtocol() string {
	return "instagram"
}

func (c *InstagramConnector) GetIcon() string {
	return "mdi:instagram"
}

func (c *InstagramConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	log.Infof("Instagram connector refresh requested for account %d (stub implementation)", socialAccount.ID)
	return nil
}

func (c *InstagramConnector) GetOAuth2Config() *oauth2.Config {
	log.Infof("Instagram OAuth2 config requested (stub implementation)")
	return nil
}

func (c *InstagramConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	log.Infof("Instagram OAuth2 callback received (stub implementation)")
	return nil
}
