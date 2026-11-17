package hooks

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httptest.NewTLSServer uses a certificate no client trusts by default, like a
// self-signed webhook endpoint.
func TestClientForVerifiesTLSByDefault(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	post := func(client *http.Client) (int, error) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
		require.NoError(t, err)
		res, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		return res.StatusCode, res.Body.Close()
	}

	_, err := post(clientFor(false))
	var certErr *tls.CertificateVerificationError
	require.ErrorAs(t, err, &certErr, "a hook must verify TLS unless it opts out")

	status, err := post(clientFor(true))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestFromDBKeepsTLSSetting(t *testing.T) {
	got := FromDB([]*db.WebhookHook{
		{URL: "https://a.example/hook", Enabled: true},
		{URL: "https://b.example/hook", Enabled: false, SkipTLSVerify: true},
	})
	assert.Equal(t, []runtimeconfig.IncomingMessageHook{
		{URL: "https://a.example/hook", Enabled: true},
		{URL: "https://b.example/hook", Enabled: false, SkipTLSVerify: true},
	}, got)
}
