package httpserver

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"

	"github.com/jamesread/japella/internal/controlapi"
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

func Start() {
	mux := http.NewServeMux()

	apipath, apihandler, srv := controlapi.GetNewHandler()

	mux.Handle("/api"+apipath, http.StripPrefix("/api", apihandler))
	mux.Handle("/oauth2callback", http.HandlerFunc(srv.OAuth2CallbackHandler))
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
