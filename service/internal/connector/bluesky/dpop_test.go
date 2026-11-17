package bluesky

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDPoPKeyStoredRoundTrip(t *testing.T) {
	original, err := generatePrivateKey()
	require.NoError(t, err)

	stored, err := encodeStoredDPoPKey(original)
	require.NoError(t, err)

	// Stored keys are the private scalar in decimal, the format existing rows use.
	raw, err := original.Bytes()
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).SetBytes(raw).String(), stored)

	loaded, err := decodeStoredDPoPKey(stored)
	require.NoError(t, err)
	assert.True(t, original.Equal(loaded))

	hash := sha256.Sum256([]byte("dpop"))
	sig, err := ecdsa.SignASN1(rand.Reader, loaded, hash[:])
	require.NoError(t, err)
	assert.True(t, ecdsa.VerifyASN1(&original.PublicKey, hash[:], sig))
}

func TestDecodeStoredDPoPKeyRejectsInvalid(t *testing.T) {
	for _, stored := range []string{"", "not-a-number", "0", "-5"} {
		_, err := decodeStoredDPoPKey(stored)
		assert.Error(t, err, "stored key %q", stored)
	}
}

// dpopServer replies with each response in turn and records what it received.
type dpopServer struct {
	responses []func(w http.ResponseWriter)
	proofs    []map[string]any
	auth      []string
	bodies    []string
}

func (d *dpopServer) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.proofs = append(d.proofs, jwtPayload(t, r.Header.Get("DPoP")))
		d.auth = append(d.auth, r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		d.bodies = append(d.bodies, string(body))
		d.responses[len(d.proofs)-1](w)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func jwtPayload(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	require.Len(t, parts, 3, "DPoP proof is a JWT")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func reply(status int, nonce, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		if nonce != "" {
			w.Header().Set("DPoP-Nonce", nonce)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func newTestTransport(t *testing.T, token string) *dpopTransport {
	t.Helper()
	key, err := generatePrivateKey()
	require.NoError(t, err)
	return &dpopTransport{base: http.DefaultTransport, privKey: key, signedAccessToken: token}
}

// roundTrip returns the status code and body; 0 when there was no response.
func roundTrip(t *testing.T, tr *dpopTransport, url, body string) (int, string, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url+"/xrpc/x", strings.NewReader(body))
	require.NoError(t, err)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(body)), nil }
	res, err := tr.RoundTrip(req)
	if res == nil {
		return 0, "", err
	}
	got, readErr := io.ReadAll(res.Body)
	require.NoError(t, readErr)
	require.NoError(t, res.Body.Close())
	return res.StatusCode, string(got), err
}

func TestDPoPTransportSignsRequestsAndKeepsNonce(t *testing.T) {
	d := &dpopServer{responses: []func(http.ResponseWriter){reply(200, "n1", "ok"), reply(200, "", "ok")}}
	url := d.start(t)
	tr := newTestTransport(t, "tok")

	status, body, err := roundTrip(t, tr, url, "payload")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "ok", body)
	assert.Equal(t, "DPoP tok", d.auth[0])
	assert.Equal(t, "POST", d.proofs[0]["htm"])
	assert.Equal(t, url+"/xrpc/x", d.proofs[0]["htu"])
	assert.Equal(t, calculateAccessTokenHash("tok"), d.proofs[0]["ath"])
	assert.NotContains(t, d.proofs[0], "nonce")

	_, _, err = roundTrip(t, tr, url, "payload")
	require.NoError(t, err)
	assert.Equal(t, "n1", d.proofs[1]["nonce"], "the server's nonce is used on the next request")
}

func TestDPoPTransportRetriesWithServerNonce(t *testing.T) {
	d := &dpopServer{responses: []func(http.ResponseWriter){
		reply(401, "n2", `{"error":"use_dpop_nonce"}`),
		reply(200, "", "done"),
	}}
	url := d.start(t)

	status, body, err := roundTrip(t, newTestTransport(t, "tok"), url, "payload")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "done", body)
	require.Len(t, d.proofs, 2)
	assert.Equal(t, "n2", d.proofs[1]["nonce"])
	assert.Equal(t, []string{"payload", "payload"}, d.bodies, "the body is sent again")
}

func TestDPoPTransportErrorResponses(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		wantErr    bool
	}{
		{"nonce error without a nonce", `{"error":"use_dpop_nonce"}`, 400, false},
		{"invalid token", `{"error":"InvalidToken"}`, 401, true},
		{"token for PDS only", `OAuth tokens are meant for PDS access only`, 400, true},
		{"other error", `{"error":"BadRequest"}`, 400, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := &dpopServer{responses: []func(http.ResponseWriter){reply(c.status, "", c.body)}}
			status, body, err := roundTrip(t, newTestTransport(t, "tok"), d.start(t), "payload")
			assert.Equal(t, c.status, status)
			assert.Equal(t, c.body, body, "the body is still readable")
			if c.wantErr {
				require.EqualError(t, err, "invalid or expired OAuth token")
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, d.proofs, 1, "no retry")
		})
	}
}

func TestDPoPTransportWithoutAccessToken(t *testing.T) {
	d := &dpopServer{responses: []func(http.ResponseWriter){reply(200, "", "ok")}}
	_, _, err := roundTrip(t, newTestTransport(t, ""), d.start(t), "")
	require.NoError(t, err)
	assert.Empty(t, d.auth[0])
	assert.NotContains(t, d.proofs[0], "ath")
}
