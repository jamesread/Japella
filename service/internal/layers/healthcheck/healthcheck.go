package healthcheck

import (
	"net/http"
	"strings"

	"github.com/jamesread/japella/internal/layers/api"
	log "github.com/sirupsen/logrus"
)

// ReadinessMiddleware serves liveness and SPA bootstrap routes without requiring
// full readiness. API/MCP and other privileged routes return HTTP 500 until the
// API reports ready (database migrated + connector controller started).
//
// GetStatus is always allowed so the frontend can show a structured database
// error state while startup is paused.
func ReadinessMiddleware(srv *api.ControlApi, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if allowWithoutReady(path) {
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

func allowWithoutReady(path string) bool {
	switch path {
	case "/healthz", "/metrics", "/llms.txt", "/lang":
		return true
	}

	// SPA shell and static assets (not API/MCP).
	if !strings.HasPrefix(path, "/api") &&
		path != "/mcp" &&
		!strings.HasPrefix(path, "/mcp/") &&
		path != "/upload" &&
		!strings.HasPrefix(path, "/media/") &&
		path != "/oauth2callback" &&
		!strings.HasPrefix(path, "/oauth/") {
		return true
	}

	// Structured status for the frontend startup gate.
	if strings.HasSuffix(path, "/JapellaControlApiService/GetStatus") {
		return true
	}

	return false
}
