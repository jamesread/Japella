package main

import (
	"github.com/jamesread/japella/internal/bots/exec"
	"github.com/jamesread/japella/internal/buildinfo"
	"github.com/jamesread/japella/internal/httpserver"
	"github.com/jamesread/japella/internal/nanoservice"
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	serviceRegistry = make(map[string]nanoservice.Nanoservice)
	Version         = "dev"
)

func main() {
	/**
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    false,
		DisableTimestamp: true,
	})
	*/

	log.SetOutput(os.Stdout)

	if os.Getenv("JAPELLA_DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.Infof("japella startup")
	log.WithFields(log.Fields{
		"version":   buildinfo.Version,
		"buildDate": buildinfo.BuildDate,
		"commit":    buildinfo.Commit,
	}).Infof("buildinfo")

	initServiceRegistry()
	startNanoservices()

	httpserver.Start()
}

func initServiceRegistry() {
	/*
	   serviceRegistry["discord"] = discord.DiscordConnector{}
	   serviceRegistry["mastodon"] = mastodon.MastodonConnector{}
	   serviceRegistry["exec"] = exec.Exec{}
	   serviceRegistry["dblogger"] = dblogger.DbLogger{}
	*/
	serviceRegistry["exec"] = &exec.Exec{}
}

func startNanoservices() {
	services := nanoservice.GetNanoservices()

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
