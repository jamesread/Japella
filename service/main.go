package main

import (
	"github.com/jamesread/japella/internal/bots/dblogger"
	"github.com/jamesread/japella/internal/bots/exec"
	"github.com/jamesread/japella/internal/connector/discord"
	"github.com/jamesread/japella/internal/connector/mastodon"
	"github.com/jamesread/japella/internal/connector/telegram"
	"github.com/jamesread/japella/internal/httpserver"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

var serviceRegistry = make(map[string]nanoservice.Nanoservice)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    false,
		DisableTimestamp: true,
	})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	log.Infof("japella startup")

	supportedVersion := 2
	configVersion := runtimeconfig.Get().ConfigVersion

	if configVersion == 0 {
		log.Fatal("The configuration version is zero, this probably means `configVersion` has not been set.")
		os.Exit(1)
	}

	if configVersion != supportedVersion {
		log.Fatalf("This version of Japella only supports config files with version %v", supportedVersion)
	}

	initServiceRegistry()
	startNanoservices()

	httpserver.Start()
}

func initServiceRegistry() {
	serviceRegistry["telegram"] = telegram.TelegramConnector{}
	serviceRegistry["discord"] = discord.DiscordConnector{}
	serviceRegistry["mastodon"] = mastodon.MastodonConnector{}
	serviceRegistry["exec"] = exec.Exec{}
	serviceRegistry["dblogger"] = dblogger.DbLogger{}
}

func startNanoservices() {
	services := getNanoservices()

	log.WithFields(log.Fields{
		"names": services,
	}).Infof("Starting nanoservices")

	for _, serviceName := range services {
		if serviceName == "" {
			continue
		}

		startService(serviceName)
	}

	log.Infof("japella started")
}

func startService(serviceName string) {
	service, ok := serviceRegistry[serviceName]

	if !ok {
		log.WithFields(log.Fields{
			"name": serviceName,
		}).Errorf("Service not found")
		return
	} else {
		log.WithFields(log.Fields{
			"name": serviceName,
		}).Infof("Starting service")
	}

	go service.Start()
}

func getNanoservices() []string {
	services := strings.Split(os.Getenv("JAPELLA_NANOSERVICES"), ",")

	return services
}
