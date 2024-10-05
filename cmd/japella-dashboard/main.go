package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/jamesread/japella/internal/dashboard"
)

func main() {
	log.Infof("japella-dasboard")

	dashboard.Start()

	// Block forever
	select {}
}
