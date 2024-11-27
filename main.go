package main

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
	"os"

	"github.com/go-kod/kod"

	"context"
)

func main() {
	log.Infof("japella")

	kod.WithConfigFile("japella.toml")

	if err := kod.Run(context.Background(), serve); err != nil {
		log.Fatalf("error: %v", err)
	}
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

		log.Infof("Starting service: %s", serviceName)
	}

	log.Infof("japella started")

	for {
		time.Sleep(1 * time.Second)
	}
}

func getNanoservices() []string {
	services := strings.Split(os.Getenv("JAPELLA_NANOSERVICES"), ",")

	return services
}
