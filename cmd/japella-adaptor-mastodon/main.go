package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/japella/internal/runtimeconfig"
	masto "github.com/jamesread/japella/internal/adaptor/mastodon"
	"time"
)

var cfg struct {
	Common *runtimeconfig.CommonConfig
	Mastodon *masto.MastodonConfig
}

func main() {
	log.Infof("japella-adaptor-mastodon")

	cfg.Common = runtimeconfig.LoadNewConfigCommon()

	runtimeconfig.LoadConfig("config.mastodon.yaml", &cfg.Mastodon)

	go masto.New(cfg.Mastodon).Start()

	for {
		time.Sleep(1 * time.Second)
	}
}
