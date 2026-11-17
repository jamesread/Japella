package hooks

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
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

// captureReplies records what executeHook sends back to the chat.
func captureReplies(t *testing.T) *[]*msgs.OutgoingMessage {
	t.Helper()
	var sent []*msgs.OutgoingMessage
	orig := publishOutgoing
	publishOutgoing = func(m *msgs.OutgoingMessage) { sent = append(sent, m) }
	t.Cleanup(func() { publishOutgoing = orig })
	return &sent
}

func hookServer(t *testing.T, status int, body string, gotBody *[]byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotBody != nil {
			*gotBody, _ = io.ReadAll(r.Body)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

var testIncoming = &msgs.IncomingMessage{Channel: "chan", Protocol: "telegram", MessageId: "m1", Identity: "bot", Author: "alice"}

func runHook(t *testing.T, url string) []*msgs.OutgoingMessage {
	t.Helper()
	sent := captureReplies(t)
	executeHook(runtimeconfig.IncomingMessageHook{URL: url, Enabled: true}, []byte(`{"x":1}`), 0, testIncoming, nil)
	return *sent
}

func TestExecuteHookReplies(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   []string
	}{
		{"string output", 200, `{"output":"hi"}`, []string{"hi"}},
		{"number output", 200, `{"output":3.5}`, []string{"3.5"}},
		{"bool output", 201, `{"output":true}`, []string{"true"}},
		{"false output", 200, `{"output":false}`, []string{"false"}},
		{"object output", 200, `{"output":{"a":1}}`, []string{`{"a":1}`}},
		{"null output", 200, `{"output":null}`, []string{"null"}},
		{"empty string output", 200, `{"output":""}`, nil},
		{"no output field", 200, `{"other":1}`, nil},
		{"invalid json", 200, `not json`, nil},
		{"empty body", 204, ``, nil},
		{"error with body", 500, `boom`, []string{"⚠️ Webhook error: Webhook returned error status 500: boom"}},
		{"error without body", 503, ``, []string{"⚠️ Webhook error: Webhook returned error status 503"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, m := range runHook(t, hookServer(t, c.status, c.body, nil)) {
				got = append(got, m.Content)
				assert.Equal(t, "chan", m.Channel)
				assert.Equal(t, "telegram", m.Protocol)
				assert.Equal(t, "m1", m.IncommingMessageId)
				assert.Equal(t, "bot", m.Identity)
				assert.Equal(t, db.BuildConversationKey("chan", "alice"), m.ConversationKey)
			}
			assert.Equal(t, c.want, got)
		})
	}
}

func TestExecuteHookSendsPayload(t *testing.T) {
	var body []byte
	runHook(t, hookServer(t, 200, ``, &body))
	assert.JSONEq(t, `{"x":1}`, string(body))
}

func TestExecuteHookTransportErrors(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	sent := runHook(t, closed.URL)
	require.Len(t, sent, 1)
	assert.Contains(t, sent[0].Content, "⚠️ Webhook error: Webhook call failed: ")

	sent = runHook(t, "http://bad host/")
	require.Len(t, sent, 1)
	assert.Contains(t, sent[0].Content, "⚠️ Webhook error: Failed to create webhook request: ")
}
