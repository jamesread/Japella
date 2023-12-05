package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/adaptor/discord"
	"time"

	"gopkg.in/yaml.v2"
)

var cfg struct {
	Amqp runtimeconfig.AmqpConfig
	AppId string
	PublicKey string
	Token string
}

func main() {
	log.Infof("japella-adaptor-discord")

	yaml.UnmarshalStrict(runtimeconfig.Load("config.yaml"), &cfg)

	amqp.AmqpHost = cfg.Amqp.Host
	amqp.AmqpUser = cfg.Amqp.User
	amqp.AmqpPass = cfg.Amqp.Pass
	amqp.AmqpPort = cfg.Amqp.Port

	log.Infof("cfg: %+v", cfg)

	discord.Start(cfg.AppId, cfg.PublicKey, cfg.Token)

	for {
		time.Sleep(1 * time.Second)
	}
}
