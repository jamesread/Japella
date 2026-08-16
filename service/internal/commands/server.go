package commands

import (
	"github.com/jamesread/japella/internal/bots/exec"
	"github.com/jamesread/japella/internal/buildinfo"
	"github.com/jamesread/japella/internal/httpserver"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

var serviceRegistry = map[string]nanoservice.Nanoservice{
	"exec": &exec.Exec{},
}

func runServer(cmd *cobra.Command, args []string) {
	log.Infof("japella startup")
	log.WithFields(log.Fields{
		"version":   buildinfo.Version,
		"buildDate": buildinfo.BuildDate,
		"commit":    buildinfo.Commit,
	}).Infof("buildinfo")

	startNanoservices()

	listenAddress := httpserver.GetListenAddress()
	log.WithFields(log.Fields{
		"listenAddress": listenAddress,
	}).Infof("japella started")

	httpserver.Start()
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
}

func startService(serviceName string) {
	service, ok := serviceRegistry[serviceName]
	if !ok {
		log.WithFields(log.Fields{
			"name": serviceName,
		}).Errorf("Service not found")
		return
	}

	log.WithFields(log.Fields{
		"name": serviceName,
	}).Infof("Starting service")

	go service.Start()
}
