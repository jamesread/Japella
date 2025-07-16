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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"os"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	// Register metrics with the default registry
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

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
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.StripPrefix("/", frontend.GetNewHandler()))

	server := &http.Server{
		Addr:    endpoint,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	return server, nil
}

func findCerts() (string, string) {
	crtPath := os.Getenv("JAPELLA_TLS_CRT_PATH")
	keyPath := os.Getenv("JAPELLA_TLS_KEY_PATH")

	if crtPath == "" || keyPath == "" {
		log.Warn("TLS_CRT_PATH or TLS_KEY_PATH environment variables not set, using HTTP instead of HTTPS")
		return "", ""
	}

	if _, err := os.Stat(crtPath); os.IsNotExist(err) {
		log.Errorf("TLS certificate file not found at %s", crtPath)
		return "", ""
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Errorf("TLS key file not found at %s", keyPath)
		return "", ""
	}
	
	log.Infof("Using TLS certificates: %s (crt), %s (key)", crtPath, keyPath)

	return crtPath, keyPath
}

func Start() {
	crt, key := findCerts()

	if crt != "" && key != "" {
		startHttpsServer(crt, key)
	} else {
		startHttpServer()
	}
}

func startHttpsServer(crt string, key string) {
	listenAddress := os.Getenv("JAPELLA_LISTEN_ADDRESS")

	if listenAddress == "" {
		listenAddress =  "0.0.0.0:443"
	}

	server, err := CreateServer(listenAddress)

	if err != nil {
		log.Errorf("Error creating server: %v", err)
		return
	}

	log.Infof("Using TLS certificates for HTTPS")

	if err := server.ListenAndServeTLS(crt, key); err != nil {
		log.Errorf("Error: %v", err)
	}
}

func startHttpServer() {
	listenAddress := os.Getenv("JAPELLA_LISTEN_ADDRESS")

	if listenAddress == "" {
		listenAddress  = "0.0.0.0:8080"
	}

	server, err := CreateServer(listenAddress)

	if err != nil {
		log.Errorf("Error creating server: %v", err)
		return
	}

	log.Infof("No TLS certificates found, using HTTP")

	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Error: %v", err)
	}
}
