package healthcheck

import (
	"net/http"

	"github.com/jamesread/japella/internal/layers/api"
	log "github.com/sirupsen/logrus"
)

// ReadinessMiddleware serves /healthz and /metrics without a database check (liveness / observability).
// All other routes, including /readyz, return HTTP 500 until the API reports ready (database + connector controller).
func ReadinessMiddleware(srv *api.ControlApi, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch path {
		case "/healthz", "/metrics":
			next.ServeHTTP(w, r)
			return
		}

		if !srv.IsReady() {
			internal := srv.ReadinessErrorMessage()
			if internal == "" {
				internal = "service not ready"
			}
			log.Debugf("Readiness: rejecting %s %s: %s", r.Method, path, internal)
			http.Error(w, srv.ReadinessClientMessage(), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
