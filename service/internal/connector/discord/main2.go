package discord

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
)

type DiscordConnector struct {
	nickname  string
	isRunning bool
	hooks     []runtimeconfig.IncomingMessageHook // Webhooks to call when messages are received
	db        *db.DB                               // Database reference for loading hooks

	utils.LogComponent
	connector.BaseConnector
	connector.ConnectorWithYamlConfig
}

func (a *DiscordConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	config, _ := startup.Config.(*runtimeconfig.DiscordConfig)

	if config == nil || config.Token == "" {
		a.Logger().Errorf("Discord bot token is not set in configuration")
		return
	}

	// Store DB reference for later hook loading (after bot starts and nickname is available)
	a.db = startup.DB
	a.hooks = config.IncomingMessageHooks
	if len(a.hooks) > 0 {
		a.Logger().Infof("Configured %d incoming message hook(s) from config (will load from DB after bot starts)", len(a.hooks))
	}

	a.StartWithToken(config.Token)
}

func (a *DiscordConnector) Start() {
	// Bot is started in SetStartupConfiguration via StartWithToken
	// This method exists to satisfy the BaseConnector interface
}

func (a *DiscordConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	// Discord uses bot tokens, not OAuth, so no refresh is needed
	return nil
}

func (a *DiscordConnector) StartWithToken(token string) {
	a.SetPrefix("Discord")
	a.Logger().Infof("Discord connector starting")

	session := a.startActual(token)

	if session == nil {
		a.Logger().Errorf("Discord session not available")
		a.isRunning = false
		return
	}

	if runtimeconfig.Get().Amqp.Enabled {
		go a.Replier()
	}

	a.Logger().Infof("Discord connector started successfully")
}
