package httpserver

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/jamesread/japella/internal/httpserver/frontend"
	"github.com/jamesread/japella/internal/httpserver/i18n"
	"github.com/jamesread/japella/internal/httpserver/upload"
	"github.com/jamesread/japella/internal/layers/api"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/layers/healthcheck"
	log "github.com/sirupsen/logrus"
)

/*
func allowCors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, X-User-Agent")

		if origin := r.Header.Get("Origin"); origin != "" {
			log.Infof("Adding CORS Header origin %v", origin)

			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		h.ServeHTTP(w, r)
	})
}
*/

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		log.Errorf("Error writing response: %v", err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("healthy")); err != nil {
		log.Errorf("Error writing response: %v", err)
	}
}

func CreateServer(endpoint string) (*http.Server, error) {
	mux := http.NewServeMux()

	apipath, apihandler, srv := api.GetNewHandler()

	healthcheckLayer := healthcheck.NewHealthCheckLayer(srv)
	healthcheckHandler := healthcheckLayer.Wrap(apihandler)

	authenticationLayer := authentication.DefaultAuthLayer(srv.DB)
	authenticatedApiHandler := authenticationLayer.WrapHandler(healthcheckHandler)

	mux.Handle("/api"+apipath, http.StripPrefix("/api", authenticatedApiHandler))
	mux.Handle("/oauth2callback", http.HandlerFunc(srv.OAuth2CallbackHandler))
	mux.HandleFunc("/oauth/client-metadata.json", srv.OAuthClientMetadataHandler)
	mux.Handle("/lang", http.HandlerFunc(i18n.Handle))
	mux.HandleFunc("/upload", upload.Handle)
	mux.HandleFunc("/readyz", handleReadyz)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/", http.StripPrefix("/", frontend.GetNewHandler()))

	server := &http.Server{
		Addr:    endpoint,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	return server, nil
}

func Start() {
	server, err := CreateServer("0.0.0.0:8080")
	if err != nil {
		log.Errorf("Error creating server: %v", err)
		return
	}

	log.Infof("Starting http server on %v", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Error: %v", err)
	}
}
