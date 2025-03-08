package main

import (
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/dashboard"
	"github.com/jamesread/japella/internal/bots/dblogger"
	"github.com/jamesread/japella/internal/connector/telegram"
	"github.com/jamesread/japella/internal/connector/discord"
	"github.com/jamesread/japella/internal/bots/exec"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
	"os"

	"github.com/go-kod/kod"

	"context"
)

var serviceRegistry = make(map[string]nanoservice.Nanoservice)

func main() {
	log.Infof("japella startup")

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: false,
		DisableTimestamp: true,
	})

	kod.WithConfigFile("japella.toml")

	if err := kod.Run(context.Background(), serve); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	serviceRegistry["telegram"] = telegram.TelegramConnector{}
	serviceRegistry["discord"] = discord.DiscordConnector{}
	serviceRegistry["dashboard"] = dashboard.Dashboard{}
	serviceRegistry["exec"] = exec.Exec{}
	serviceRegistry["dblogger"] = dblogger.DbLogger{}
}

type app struct {
	kod.Implements[kod.Main]
}

func serve(context.Context, *app) error {
	startNanoservices()
	return nil
}

func startNanoservices() {
	services := getNanoservices()

	log.WithFields(log.Fields {
		"names": services,
	}).Infof("Starting nanoservices")

	for _, serviceName := range services {
		if serviceName == "" {
			continue
		}

		startService(serviceName)
	}

	startService("dashboard")

	log.Infof("japella started")

	for {
		time.Sleep(1 * time.Second)
	}
}

func startService(serviceName string) {
	service, ok := serviceRegistry[serviceName]

	if !ok {
		log.WithFields(log.Fields {
			"name": serviceName,
		}).Errorf("Service not found")
		return
	} else {
		log.WithFields(log.Fields {
			"name": serviceName,
		}).Infof("Starting service")
	}

	go service.Start()
}

func getNanoservices() []string {
	services := strings.Split(os.Getenv("JAPELLA_NANOSERVICES"), ",")

	return services
}
