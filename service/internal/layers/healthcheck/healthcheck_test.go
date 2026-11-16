package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowWithoutReady(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/metrics", true},
		{"/llms.txt", true},
		{"/lang", true},
		{"/", true},
		{"/assets/index.js", true},
		{"/api/japella.controlapi.v1.JapellaControlApiService/GetStatus", true},
		{"/api/japella.controlapi.v1.JapellaControlApiService/GetStatusX", false},
		{"/api/japella.controlapi.v1.JapellaControlApiService/SubmitPost", false},
		{"/mcp", false},
		{"/mcp/", false},
		{"/upload", false},
		{"/media/files/x.png", false},
		{"/oauth2callback", false},
		{"/oauth/client-metadata.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, allowWithoutReady(tt.path))
		})
	}
}
