package facebook

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type FacebookConnector struct{}

func (c *FacebookConnector) SetStartupConfiguration(*connector.ControllerStartupConfiguration) {}

func (c *FacebookConnector) Start() {
	log.Infof("Facebook connector started (stub implementation)")
}

func (c *FacebookConnector) GetIdentity() string {
	return "Facebook"
}

func (c *FacebookConnector) GetProtocol() string {
	return "facebook"
}

func (c *FacebookConnector) GetIcon() string {
	return "mdi:facebook"
}

func (c *FacebookConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	log.Infof("Facebook connector refresh requested for account %d (stub implementation)", socialAccount.ID)
	return nil
}

func (c *FacebookConnector) GetOAuth2Config() *oauth2.Config {
	log.Infof("Facebook OAuth2 config requested (stub implementation)")
	return nil
}

func (c *FacebookConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	log.Infof("Facebook OAuth2 callback received (stub implementation)")
	return nil
}
