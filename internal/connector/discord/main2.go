package discord

import (
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"time"
)

type DiscordConnector struct {
	utils.LogComponent
}

func (a DiscordConnector) Start() {
	a.SetPrefix("Discord")
	a.Logger().Infof("Discord connector started")

	cfg := runtimeconfig.Get()

	session := startActual(cfg.Connectors.Discord.AppId, cfg.Connectors.Discord.PublicKey, cfg.Connectors.Discord.Token)

	if session == nil {
		a.Logger().Errorf("Discord session not available")
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
