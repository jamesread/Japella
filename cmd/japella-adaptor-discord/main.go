package main

import (
	"github.com/jamesread/japella/internal/adaptor/discord"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
	"time"
)

var cfg struct {
	Common  *runtimeconfig.CommonConfig
	Discord struct {
		AppId     string
		PublicKey string
		Token     string
	}
}

func main() {
	log.Infof("japella-adaptor-discord")

	cfg.Common = runtimeconfig.LoadNewConfigCommon()
	runtimeconfig.LoadConfig("config.discord.yaml", &cfg.Discord)

	log.WithFields(log.Fields{
		"amqpHost":     cfg.Common.Amqp.Host,
		"appId":     cfg.Discord.AppId,
	}).Infof("cfg after parse")

	discord.Start(cfg.Discord.AppId, cfg.Discord.PublicKey, cfg.Discord.Token)

	for {
		time.Sleep(1 * time.Second)
	}
}
