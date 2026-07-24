package httpserver

import (
	_ "embed"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/jamesread/japella/internal/httpserver/frontend"
	"github.com/jamesread/japella/internal/httpserver/i18n"
	"github.com/jamesread/japella/internal/httpserver/upload"
	"github.com/jamesread/japella/internal/layers/api"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/layers/healthcheck"
	"github.com/jamesread/japella/internal/mcp"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/shutdown"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

//go:embed llms.txt
var llmsTxt []byte

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

	authenticationLayer := authentication.DefaultAuthLayer(srv.DB)
	authenticatedApiHandler := authenticationLayer.WrapHandler(apihandler)
	authenticatedMCPHandler := authenticationLayer.WrapMCPHandler(mcp.NewHandler(srv))

	mux.Handle("/api"+apipath, http.StripPrefix("/api", authenticatedApiHandler))
	mux.Handle("/oauth2callback", http.HandlerFunc(srv.OAuth2CallbackHandler))
	mux.HandleFunc("/oauth/client-metadata.json", srv.OAuthClientMetadataHandler)
	mux.Handle("/lang", http.HandlerFunc(i18n.Handle))
	mux.HandleFunc("/upload", upload.Handle)
	mux.HandleFunc("/media/files/", upload.HandleServeMedia)
	mux.HandleFunc("/readyz", handleReadyz)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/llms.txt", handleLLMsTxt)
	mux.Handle("/mcp", authenticatedMCPHandler)
	mux.Handle("/mcp/", authenticatedMCPHandler)
	mux.Handle("/", http.StripPrefix("/", frontend.GetNewHandler()))

	handler := healthcheck.ReadinessMiddleware(srv, mux)

	server := &http.Server{
		Addr:    endpoint,
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}

	return server, nil
}

func handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(llmsTxt); err != nil {
		log.Errorf("Error writing llms.txt: %v", err)
	}
}

func findCerts() (string, string) {
	cfg := runtimeconfig.Get()
	crtPath := cfg.TLS.CrtPath
	keyPath := cfg.TLS.KeyPath

	if crtPath == "" || keyPath == "" {
		log.Warn("TLS crtPath or keyPath not set in config, using HTTP instead of HTTPS")
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

// GetListenAddress returns the effective listen address that will be used.
// Priority: $PORT, then config listenAddress, then TLS :443 or default :8080.
func GetListenAddress() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return "0.0.0.0:" + port
	}

	cfg := runtimeconfig.Get()
	if cfg.ListenAddress != "" {
		return cfg.ListenAddress
	}

	crt, key := findCerts()
	if crt != "" && key != "" {
		return "0.0.0.0:443"
	}

	return "0.0.0.0:8080"
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
	listenAddress := GetListenAddress()

	server, err := CreateServer(listenAddress)

	if err != nil {
		log.Errorf("Error creating server: %v", err)
		return
	}

	log.Infof("Using TLS certificates for HTTPS")

	serve(server, true, crt, key)
}

func startHttpServer() {
	listenAddress := GetListenAddress()

	server, err := CreateServer(listenAddress)

	if err != nil {
		log.Errorf("Error creating server: %v", err)
		return
	}

	log.Infof("No TLS certificates found, using HTTP")

	serve(server, false, "", "")
}

func serve(server *http.Server, useTLS bool, crt, key string) {
	shutdown.RegisterServer(server)

	var err error
	if useTLS {
		err = server.ListenAndServeTLS(crt, key)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		log.Errorf("Error: %v", err)
	}
}
