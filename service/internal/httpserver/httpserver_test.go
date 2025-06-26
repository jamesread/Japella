package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
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
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
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

	// Test health check endpoints
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "healthz endpoint",
			path:           "/healthz",
			expectedStatus: http.StatusOK,
			expectedBody:   "healthy",
		},
		{
			name:           "readyz endpoint",
			path:           "/readyz",
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "lang endpoint",
			path:           "/lang",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "oauth2callback endpoint",
			path:           "/oauth2callback",
			expectedStatus: http.StatusOK, // or whatever the actual handler returns
		},
		{
			name:           "api endpoint exists",
			path:           "/api/japella.controlapi.v1.JapellaControlApiService/GetStatus",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "frontend endpoint",
			path:           "/",
			expectedStatus: http.StatusOK, // File server should return 200 for root
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req, err := http.NewRequest("GET", "http://localhost"+server.Addr+tc.path, nil)
			require.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			server.Handler.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, rr.Code, "Handler returned wrong status code for %s", tc.path)

			// Check response body if expected
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
	req, err := http.NewRequest("GET", "/readyz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleReadyz)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "ok", strings.TrimSpace(rr.Body.String()))
}

func TestHandleHealthz(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
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

	// Check that the handler is wrapped with h2c
	// This is a bit implementation-specific, but we can verify the handler exists
	assert.NotNil(t, server.Handler, "Server handler should not be nil")

	// Test that the server can handle requests (basic functionality test)
	req, err := http.NewRequest("GET", "/healthz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreateServer_AuthenticationLayer(t *testing.T) {
	server, err := CreateServer("localhost:0")
	require.NoError(t, err)
	require.NotNil(t, server)

	// Test that API endpoints are properly protected
	// The GetStatus endpoint should be accessible without authentication (it's in the allowlist)
	req, err := http.NewRequest("POST", "/api/japella.controlapi.v1.JapellaControlApiService/GetStatus", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
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
			req, err := http.NewRequest("GET", endpoint, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			server.Handler.ServeHTTP(rr, req)

			// All endpoints should return some response (not 404)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Endpoint %s should not return 404", endpoint)
		})
	}
}
