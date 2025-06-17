package httpserver

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"

	"github.com/jamesread/japella/internal/layers/auth"
	"github.com/jamesread/japella/internal/layers/api"
	"github.com/jamesread/japella/internal/httpserver/i18n"
	"github.com/jamesread/japella/internal/httpserver/frontend"
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

func Start() {
	mux := http.NewServeMux()

	apipath, apihandler, srv := api.GetNewHandler()

	authLayer := auth.DefaultAuthLayer(srv.DB)
	authenticatedApiHandler := authLayer.WrapHandler(apihandler)

	mux.Handle("/api"+apipath, http.StripPrefix("/api", authenticatedApiHandler))
	mux.Handle("/oauth2callback", http.HandlerFunc(srv.OAuth2CallbackHandler))
	mux.Handle("/lang", http.HandlerFunc(i18n.Handle))
	mux.HandleFunc("/readyz", handleReadyz)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/", http.StripPrefix("/", frontend.GetNewHandler()))

	endpoint := "0.0.0.0:8080"

	log.Infof("Starting http server on %v", endpoint)
	log.Infof("API available at http://%v/api%v", endpoint, apipath)

	if err := http.ListenAndServe(
		endpoint,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		log.Errorf("Error: %v", err)
	}

}
