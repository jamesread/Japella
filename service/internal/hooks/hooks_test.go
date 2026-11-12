package hooks

import (
	"testing"

	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/stretchr/testify/assert"
)

func TestClientForHookVerifiesTLSByDefault(t *testing.T) {
	assert.Same(t, verifyingClient, clientForHook(runtimeconfig.IncomingMessageHook{URL: "https://example.org"}))
}

func TestClientForHookInsecureOptIn(t *testing.T) {
	assert.Same(t, insecureClient, clientForHook(runtimeconfig.IncomingMessageHook{URL: "https://example.org", InsecureSkipVerify: true}))
}
