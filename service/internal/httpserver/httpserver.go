package httpserver

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"

	"github.com/jamesread/japella/internal/controlapi"
	"github.com/jamesread/japella/internal/dashboard"
	log "github.com/sirupsen/logrus"
)

func Start() {
	mux := http.NewServeMux()

	apipath, apihandler := controlapi.GetNewHandler()

	mux.Handle("/api"+apipath, http.StripPrefix("/api", apihandler))
	mux.Handle("/", http.StripPrefix("/", dashboard.GetNewHandler()))

	endpoint := "localhost:8080"

	log.Infof("Starting http server on %v", endpoint)
	log.Infof("API available at http://%v/api%v", endpoint, apipath)

	if err := http.ListenAndServe(
		endpoint,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		log.Errorf("Error: %v", err)
	}

}
