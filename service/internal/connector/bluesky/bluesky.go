package bluesky

/**
The bluesky OAuth2 protocol is pretty wild, and needs a lot more work
than Mastodon or X (Twitter) to get working.

Useful links:
https://bsky.social/.well-known/oauth-authorization-server
*/

import (
	"crypto/elliptic"
	"database/sql"
	"fmt"
	"math/big"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	"github.com/jamesread/japella/internal/utils/dateutil"

	"crypto/ecdsa"

	"context"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

type BlueskyConnector struct {
	connector.BaseConnector
	connector.OAuth2Connector
	connector.ConnectorWithWall
	connector.ConfigProvider
	connector.OAuth2ConnectorWithClientRegistration

	db *db.DB
	utils.LogComponent

	tmpClient *utils.ChainingHttpClient
}

const PAR_ENDPOINT = "https://bsky.social/oauth/par"
const CFG_BSKY_CLIENT_ID = "bluesky.client_id"         // notsecret
const CFG_BSKY_CLIENT_SECRET = "bluesky.client_secret" // notsecret

// BlueSky does dynamic client registration, so we don't need to store the client ID and secret
func (b *BlueskyConnector) IsRegistered() bool {
	return true
}

func (b *BlueskyConnector) RegisterClient() error {
	return nil
}

func (b *BlueskyConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	b.db = startup.DB
}

func (b *BlueskyConnector) Start() {
	b.SetPrefix("BlueskyConnector")
}

func (b *BlueskyConnector) GetIdentity() string {
	return "untitled-account"
}

func (b *BlueskyConnector) GetProtocol() string {
	return "bluesky"
}

type BlueskyCreatePostRequest struct {
	Repo       string            `json:"repo"`
	Collection string            `json:"collection"`
	Record     BlueskyPostRecord `json:"record"`
}

type BlueskyPostRecord struct {
	Type      string `json:"$type"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type BlueskyCreatePostResponse struct {
	Uri string `json:"uri"`
	Cid string `json:"cid"`
}

func (b *BlueskyConnector) PostToWall(socialAccount *connector.SocialAccount, content string) *connector.PostResult {
	req := BlueskyCreatePostRequest{
		Repo:       socialAccount.Did,
		Collection: "app.bsky.feed.post",
		Record: BlueskyPostRecord{
			Type:      "app.bsky.feed.post",
			Text:      content,
			CreatedAt: dateutil.GetCurrentTimeRFC3339(),
		},
	}

	res := BlueskyCreatePostResponse{}

	client := b.tmpClient
	if client == nil {
		// Attempt to reconstruct a DPoP-enabled client from stored credentials
		if sa, err := b.db.GetSocialAccount(socialAccount.Id); err == nil {
			client = b.ensureDPoPClient(sa)
			b.tmpClient = client
		}
	}
	if client == nil {
		b.Logger().Errorf("Bluesky HTTP client not initialized; OAuth2 flow not completed")
		return &connector.PostResult{Err: fmt.Errorf("bluesky http client not initialized"), URL: ""}
	}
	b.Logger().Infof("Posting to Bluesky with request: %+v %+v", b, req)

	base := socialAccount.Homeserver
	if base == "" {
		base = "https://bsky.social"
	}
	endpoint := base + "/xrpc/com.atproto.repo.createRecord"
	client.PostWithJson(endpoint, &req)
	client.AsJson(&res)

	b.Logger().Infof("Response from Bluesky: %+v", res)

	// Build a web URL from the at:// URI: at://<repo>/app.bsky.feed.post/<rkey>
	rkey := res.Uri
	if idx := strings.LastIndex(rkey, "/"); idx >= 0 && idx+1 < len(rkey) {
		rkey = rkey[idx+1:]
	}
	ret := &connector.PostResult{Err: nil, URL: "https://bsky.app/profile/" + socialAccount.Identity + "/post/" + rkey}

	return ret
}

func (b *BlueskyConnector) GetIcon() string {
	return "bi:bluesky"
}

func (b *BlueskyConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	user := socialAccount.Identity
	url := "https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=" + user

	client := b.tmpClient
	if client == nil {
		client = utils.NewClient(b.Logger())
	}

	client.Get(url).AsJson(nil)

	return nil
}

func (b *BlueskyConnector) GetOAuth2Config() *oauth2.Config {
	ep := oauth2.Endpoint{
		AuthURL:  "https://bsky.social/oauth/authorize",
		TokenURL: "https://bsky.social/oauth/token",
	}

	return &oauth2.Config{
		Endpoint:     ep,
		ClientID:     b.db.GetCvarString(db.CvarKeys.BaseUrl) + "/oauth/client-metadata.json",
		ClientSecret: "",
		Scopes:       []string{"atproto"},
		RedirectURL:  b.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL),
	}
}

func (b *BlueskyConnector) GetCvars() map[string]*db.Cvar {
	return map[string]*db.Cvar{
		"bluesky.client_id": &db.Cvar{
			KeyName:      CFG_BSKY_CLIENT_ID,
			DefaultValue: "",
			Title:        "Bluesky Client ID",
			Description:  "Client ID for Bluesky OAuth2",
			Category:     "Bluesky",
			Type:         "text",
		},
		"bluesky.client_secret": &db.Cvar{
			KeyName:      CFG_BSKY_CLIENT_SECRET,
			DefaultValue: "",
			Title:        "Bluesky Client Secret",
			Description:  "Client Secret for Bluesky OAuth2",
			Category:     "Bluesky",
			Type:         "password",
		},
	}
}

func (b *BlueskyConnector) CheckConfiguration() error {
	return nil
}

func findHeader(res *http.Response) string {
	if res == nil || res.Header == nil {
		return ""
	}

	return res.Header.Get("Dpop-Nonce")
}

type dpopTransport struct {
	base              http.RoundTripper
	dpopJWT           string
	dpopNonce         string
	signedAccessToken string
	privKey           *ecdsa.PrivateKey
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dpopJwt, _ := createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce)

	req.Header.Set("DPoP", dpopJwt)
	if t.signedAccessToken != "" {
		req.Header.Set("Authorization", "DPoP "+t.signedAccessToken)
	}
	resp, err := t.base.RoundTrip(req)
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		// Try to refresh nonce and retry once
		if n := resp.Header.Get("DPoP-Nonce"); n != "" {
			t.dpopNonce = n
		} else if n := resp.Header.Get("Dpop-Nonce"); n != "" {
			t.dpopNonce = n
		}
		// Recreate DPoP with new nonce and resend (safe for idempotent POST in our usage)
		dpopJwt, _ = createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce)
		req2 := req.Clone(req.Context())
		if req.Body != nil && req.GetBody != nil {
			if bodyCopy, gerr := req.GetBody(); gerr == nil {
				req2.Body = bodyCopy
			}
		}
		req2.Header.Set("DPoP", dpopJwt)
		if t.signedAccessToken != "" {
			req2.Header.Set("Authorization", "DPoP "+t.signedAccessToken)
		}
		resp, err = t.base.RoundTrip(req2)
	}
	if resp != nil {
		// Capture nonce for next requests; accept either header capitalization
		if n := resp.Header.Get("DPoP-Nonce"); n != "" {
			t.dpopNonce = n
		} else if n := resp.Header.Get("Dpop-Nonce"); n != "" {
			t.dpopNonce = n
		}
	}
	return resp, err
}

func (b *BlueskyConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	privKey, err := generatePrivateKey()
	if err != nil {
		b.Logger().Errorf("Error creating DPoP key: %v", err)
		return err
	}

	client := utils.NewClient(b.Logger())
	tp := &dpopTransport{
		base:    client.UnderlyingClient().Transport,
		privKey: privKey,
	}
	client.UnderlyingClient().Transport = tp
	b.tmpClient = client

	config := b.GetOAuth2Config()
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.UnderlyingClient())

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		b.Logger().Infof("Error exchanging code: %v", err)
		return err
	}
	// Bind DPoP transport to access token
	tp.signedAccessToken = token.AccessToken

	b.Logger().Debugf("Received token on exchange: %+v", token)

	// Try to discover handle and DID for presentation
	handle := ""
	if h, _ := b.getSessionHandle(); h != "" {
		handle = h
	}
	did := ""
	if handle != "" {
		if d, derr := b.resolveDID(handle); derr == nil {
			did = d
		}
	}

	err = b.db.RegisterAccount(&db.SocialAccount{
		Connector:          "bluesky",
		Identity:           handle,
		Did:                did,
		Homeserver:         "https://bsky.social",
		Active:             true,
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
		DpopKey:            sql.NullString{String: privKey.D.String(), Valid: true},
	})

	return err
}

// ensureDPoPClient reconstructs a DPoP-enabled HTTP client from stored DB fields
func (b *BlueskyConnector) ensureDPoPClient(sa *db.SocialAccount) *utils.ChainingHttpClient {
	if sa == nil || !sa.DpopKey.Valid || sa.OAuth2Token == "" {
		return nil
	}
	// Rebuild private key from decimal string
	d := new(big.Int)
	if _, ok := d.SetString(sa.DpopKey.String, 10); !ok {
		return nil
	}
	priv := &ecdsa.PrivateKey{D: d, PublicKey: ecdsa.PublicKey{Curve: elliptic.P256()}}
	// Derive public key from D
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d.Bytes())

	client := utils.NewClient(b.Logger())
	tp := &dpopTransport{base: client.UnderlyingClient().Transport, privKey: priv, signedAccessToken: sa.OAuth2Token}
	client.UnderlyingClient().Transport = tp
	return client
}

type sessionResponse struct {
	Did    string `json:"did"`
	Handle string `json:"handle"`
}

// getSessionHandle fetches the current session and returns handle
func (b *BlueskyConnector) getSessionHandle() (string, error) {
	client := b.tmpClient
	if client == nil {
		return "", fmt.Errorf("client not initialized")
	}
	// GET com.atproto.server.getSession
	var sr sessionResponse
	client.Get("https://bsky.social/xrpc/com.atproto.server.getSession").AsJson(&sr)
	if client.Err != nil {
		return "", client.Err
	}
	return sr.Handle, nil
}

type resolveHandleResponse struct {
	Did string `json:"did"`
}

// resolveDID returns DID for a given handle
func (b *BlueskyConnector) resolveDID(handle string) (string, error) {
	client := b.tmpClient
	if client == nil {
		return "", fmt.Errorf("client not initialized")
	}
	url := "https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=" + handle
	var rh resolveHandleResponse
	client.Get(url).AsJson(&rh)
	if client.Err != nil {
		return "", client.Err
	}
	return rh.Did, nil
}
