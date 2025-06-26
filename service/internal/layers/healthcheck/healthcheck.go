package healthcheck

import (
	"net/http"
	"github.com/jamesread/japella/internal/layers/api"
	log "github.com/sirupsen/logrus"
)

type HealthCheckLayer struct {
	srv *api.ControlApi
}

func NewHealthCheckLayer(srv *api.ControlApi) *HealthCheckLayer {
	return &HealthCheckLayer{srv: srv}
}

func (h *HealthCheckLayer) Wrap(apiHandler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.srv.DB.GetErrorMessage() != "" {
			log.Errorf("Healthcheck database error: %s", h.srv.DB.GetErrorMessage())

			http.Error(w, h.srv.DB.GetErrorMessage(), http.StatusInternalServerError)
			return
		}

		apiHandler.ServeHTTP(w, r)
	})
}
