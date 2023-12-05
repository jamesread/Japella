package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"time"
)

var cfg struct {
	Common *runtimeconfig.CommonConfig
	Discord struct {
		AppId string
		PublicKey string
		Token string
	}
}

func main() {
	log.Infof("japella-adaptor-discord")

	runtimeconfig.LoadConfigCommon(cfg.Common)
	runtimeconfig.LoadConfig("config.discord.yaml", cfg.Discord)

	log.Infof("cfg: %+v", cfg)

	Start(cfg.Discord.AppId, cfg.Discord.PublicKey, cfg.Discord.Token)

	for {
		time.Sleep(1 * time.Second)
	}
}
