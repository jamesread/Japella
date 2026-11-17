package httpserver

import (
	"bytes"
	"context"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServer(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "valid endpoint",
			endpoint: "localhost:8080",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  false, // http.Server accepts empty addr
		},
		{
			name:     "custom port",
			endpoint: "127.0.0.1:9090",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := CreateServer(tt.endpoint)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
				assert.Equal(t, tt.endpoint, server.Addr)
				assert.NotNil(t, server.Handler)
			}
		})
	}
}

func TestCreateServer_RouteRegistration(t *testing.T) {
	server, err := CreateServer("localhost:0") // Use port 0 for testing
	require.NoError(t, err)
	require.NotNil(t, server)

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Route registration / readiness policy. Some gated routes return 200 when the
	// local test environment has a working DB (CreateServer starts ControlApi reconnect).
	testCases := []struct {
		name            string
		path            string
		allowedStatuses []int
		expectedBody    string
	}{
		{
			name:            "healthz endpoint",
			path:            "/healthz",
			allowedStatuses: []int{http.StatusOK},
			expectedBody:    "healthy",
		},
		{
			name:            "readyz endpoint",
			path:            "/readyz",
			allowedStatuses: []int{http.StatusOK, http.StatusInternalServerError},
		},
		{
			name:            "lang endpoint",
			path:            "/lang",
			allowedStatuses: []int{http.StatusOK}, // allowed while not ready for SPA bootstrap
		},
		{
			name:            "oauth2callback endpoint",
			path:            "/oauth2callback",
			allowedStatuses: []int{http.StatusOK, http.StatusInternalServerError}, // gated until ready
		},
		{
			name:            "api GetStatus allowed while not ready",
			path:            "/api/japella.controlapi.v1.JapellaControlApiService/GetStatus",
			allowedStatuses: []int{http.StatusMethodNotAllowed}, // Connect expects POST
		},
		{
			name:            "frontend endpoint",
			path:            "/",
			allowedStatuses: []int{http.StatusOK}, // SPA shell allowed while not ready
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost"+server.Addr+tc.path, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			server.Handler.ServeHTTP(rr, req)

			assert.Contains(t, tc.allowedStatuses, rr.Code, "Handler returned unexpected status for %s (body: %q)", tc.path, rr.Body.String())

			if tc.expectedBody != "" {
				assert.Equal(t, tc.expectedBody, strings.TrimSpace(rr.Body.String()), "Handler returned unexpected body for %s", tc.path)
			}
		})
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Error shutting down server: %v", err)
	}
}

func TestHandleReadyz(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleReadyz)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "ok", strings.TrimSpace(rr.Body.String()))
}

func TestHandleHealthz(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleHealthz)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "healthy", strings.TrimSpace(rr.Body.String()))
}

func TestCreateServer_HTTP2Support(t *testing.T) {
	server, err := CreateServer("localhost:0")
	require.NoError(t, err)
	require.NotNil(t, server)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	for _, proto := range []struct {
		name  string
		set   func(*http.Protocols)
		major int
	}{
		{"HTTP/1.1", func(p *http.Protocols) { p.SetHTTP1(true) }, 1},
		// Connect clients use cleartext HTTP/2 with prior knowledge.
		{"cleartext HTTP/2", func(p *http.Protocols) { p.SetUnencryptedHTTP2(true) }, 2},
	} {
		t.Run(proto.name, func(t *testing.T) {
			var protocols http.Protocols
			proto.set(&protocols)
			client := &http.Client{Transport: &http.Transport{Protocols: &protocols}}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String()+"/healthz", nil)
			require.NoError(t, err)
			res, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, res.Body.Close()) })

			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, proto.major, res.ProtoMajor)
		})
	}
}

func TestCreateServer_AuthenticationLayer(t *testing.T) {
	server, err := CreateServer("localhost:0")
	require.NoError(t, err)
	require.NotNil(t, server)

	// Test that API endpoints are properly protected
	// The GetStatus endpoint should be accessible without authentication (it's in the allowlist)
	// and bypasses the readiness gate, so it succeeds whether or not the database is up.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/japella.controlapi.v1.JapellaControlApiService/GetStatus", strings.NewReader("{}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "GetStatus should be reachable unauthenticated (body: %q)", rr.Body.String())
}

func TestCreateServer_EndpointConfiguration(t *testing.T) {
	testEndpoint := "127.0.0.1:12345"
	server, err := CreateServer(testEndpoint)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, testEndpoint, server.Addr, "Server should be configured with the provided endpoint")
}

func TestCreateServer_HandlerChain(t *testing.T) {
	server, err := CreateServer("localhost:0")
	require.NoError(t, err)
	require.NotNil(t, server)

	// Test that all expected routes are registered by checking a few key endpoints
	endpoints := []string{
		"/healthz",
		"/readyz",
		"/lang",
		"/oauth2callback",
		"/api/japella.controlapi.v1.JapellaControlApiService/GetStatus",
		"/",
	}

	for _, endpoint := range endpoints {
		t.Run("endpoint_"+endpoint, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, endpoint, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			server.Handler.ServeHTTP(rr, req)

			// All endpoints should return some response (not 404)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Endpoint %s should not return 404", endpoint)
		})
	}
}

// The readiness gate answers 500 until the database is up, so this goes to the
// router directly to test the route's own authentication.
func TestUploadRequiresAuthentication(t *testing.T) {
	mediaDir := filepath.Join(t.TempDir(), "media")
	t.Chdir(filepath.Dir(mediaDir))

	router, _, err := newRouter()
	require.NoError(t, err)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "anon.html")
	require.NoError(t, err)
	_, err = part.Write([]byte("<script>alert(1)</script>"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	entries, _ := os.ReadDir(mediaDir)
	assert.Empty(t, entries, "an unauthenticated upload must not be stored")
}
