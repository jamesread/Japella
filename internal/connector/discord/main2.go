package discord

import (
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"time"
)

type DiscordConnector struct {
	log utils.LogComponent
}

func (a DiscordConnector) Start() {
	a.log.SetPrefix("Discord")
	a.log.Logger().Infof("Discord connector started")

	cfg := runtimeconfig.Get()

	startActual(cfg.Connectors.Discord.AppId, cfg.Connectors.Discord.PublicKey, cfg.Connectors.Discord.Token)

	for {
		time.Sleep(1 * time.Second)
	}
}
