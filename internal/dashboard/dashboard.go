package dashboard

import (
	"net/http"
	//	"net/http/httputil"
	//	"net/url"
	log "github.com/sirupsen/logrus"
)

type Dashboard struct {
}

func (d Dashboard) Start() {
	log.Info("Starting Dashboard")

	go StartGrpc()
	go StartRestGateway()
	go StaticFileServer()
}

func StaticFileServer() {
	listenAddress := ":8080"

	log.Infof("Starting webui on %v", listenAddress)

	http.Handle("/", http.FileServer(http.Dir("./webui")))
	err := http.ListenAndServe(listenAddress, nil)

	if err != nil {
		log.Errorf("Error starting static file server %s", err)
	}
}
