package discord

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/utils"
	"time"
)

type DiscordConnector struct {
	nickname string

	utils.LogComponent
	connector.BaseConnector
}

func (a *DiscordConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	//	config, _ := startup.Config.(*runtimeconfig.DiscordConfig)

	// a.Start(config.Token)
}

func (a *DiscordConnector) Start() {}

func (a *DiscordConnector) StartWithToken(token string) {
	a.SetPrefix("Discord")
	a.Logger().Infof("Discord connector started")

	session := a.startActual(token)

	if session == nil {
		a.Logger().Errorf("Discord session not available")
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
