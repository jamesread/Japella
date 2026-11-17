package healthcheck

import (
	"net/http"
	"strings"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
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
	if isPublicSPAPath(path) {
		return true
	}
	// Structured status for the frontend startup gate.
	return path == "/api"+controlv1.JapellaControlApiServiceGetStatusProcedure
}

var exactPrivilegedPaths = map[string]bool{
	"/mcp":            true,
	"/upload":         true,
	"/oauth2callback": true,
}

var privilegedPrefixes = []string{"/api", "/mcp/", "/media/", "/oauth/"}

// isPublicSPAPath is true for the SPA shell and static assets (not API/MCP).
func isPublicSPAPath(path string) bool {
	return !isPrivilegedPath(path)
}

func isPrivilegedPath(path string) bool {
	if exactPrivilegedPaths[path] {
		return true
	}
	for _, prefix := range privilegedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
