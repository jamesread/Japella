package bluesky

/**
The bluesky OAuth2 protocol is pretty wild, and needs a lot more work
than Mastodon or X (Twitter) to get working.

Useful links:
https://bsky.social/.well-known/oauth-authorization-server
*/

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	"github.com/jamesread/japella/internal/utils/dateutil"
	"database/sql"

	"crypto/ecdsa"

	"net/http"
	"context"

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
const CFG_BSKY_CLIENT_ID = "bluesky.client_id" // notsecret
const CFG_BSKY_CLIENT_SECRET = "bluesky.client_secret" // notsecret

// BlueSky does dynamic client registration, so we don't need to store the client ID and secret
func (b *BlueskyConnector) IsRegistered() bool {
	return true;
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
	Repo string `json:"repo"`
	Collection string `json:"collection"`
	Record BlueskyPostRecord `json:"record"`
}

type BlueskyPostRecord struct {
	Type string `json:"$type"`
	Text string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type BlueskyCreatePostResponse struct {
	Uri string `json:"uri"`
	Cid string `json:"cid"`
}

func (b *BlueskyConnector) PostToWall(socialAccount *connector.SocialAccount, content string) *connector.PostResult {
	req := BlueskyCreatePostRequest{
		Repo:       socialAccount.Identity,
		Collection: "app.bsky.feed.post",
		Record: BlueskyPostRecord{
			Type:      "app.bsky.feed.post",
			Text:      content,
			CreatedAt: dateutil.GetCurrentTimeRFC3339(),
		},
	}

	res := BlueskyCreatePostResponse{}

	client := b.tmpClient
	b.Logger().Infof("Posting to Bluesky with request: %+v %+v", b, req)

	endpoint := socialAccount.Homeserver + "/xrpc/com.atproto.repo.createRecord"
	client.PostWithJson(endpoint, &req)
	client.AsJson(res)

	b.Logger().Infof("Response from Bluesky: %+v", res)

	ret := &connector.PostResult{
		Err: nil,
		URL: "https://bsky.social/profile/" + socialAccount.Identity + "/post/" + res.Uri,
	}

	return ret
}

func (b *BlueskyConnector) GetIcon() string {
	return "bi:bluesky"
}

func (b *BlueskyConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	user := socialAccount.Identity
	url := "https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=" + user

	b.tmpClient.Get(url).AsJson(nil)

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
	base    http.RoundTripper
	dpopJWT string
	dpopNonce string
	signedAccessToken string
	privKey *ecdsa.PrivateKey
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dpopJwt, _ := createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce)

	req.Header.Set("DPoP", dpopJwt)
	req.Header.Set("Authorization", "DPoP "+t.signedAccessToken)

	return t.base.RoundTrip(req)
}

func (b *BlueskyConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {	
	privKey, err := generatePrivateKey()

	client := utils.NewClient()
	client.PostWithJson(PAR_ENDPOINT, "").AsJson(nil)

	dpopServerKey := findHeader(client.Res)

	config := b.GetOAuth2Config()

	if err != nil {
		b.Logger().Errorf("Error creating DPoP proof: %v", err)
		return err
	}

	tp := &dpopTransport{
		base:    client.UnderlyingClient().Transport,
		dpopNonce: dpopServerKey,
		privKey: privKey,
	}
	
	client.UnderlyingClient().Transport = tp 

	b.tmpClient = client

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.UnderlyingClient())

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))

	tp.signedAccessToken = token.AccessToken // Switch to using the signed token as the DPoP key
	
	if err != nil {
		b.Logger().Infof("Error exchanging code: %v", err)
		return err
	}

	b.Logger().Debugf("Received token on exchange: %+v", token)

	err = b.db.RegisterAccount(&db.SocialAccount{
		Connector:          "bluesky",
		Active:		        true,
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
		DpopKey:            sql.NullString{String: privKey.D.String(), Valid: true},
	})

	return err
}
