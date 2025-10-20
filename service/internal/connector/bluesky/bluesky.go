package bluesky

/**
The bluesky OAuth2 protocol is pretty wild, and needs a lot more work
than Mastodon or X (Twitter) to get working.

Useful links:
https://bsky.social/.well-known/oauth-authorization-server
*/

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
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
	// Get the full social account from database to access OAuth fields
	dbSocialAccount, err := b.db.GetSocialAccount(socialAccount.Id)
	if err != nil {
		b.Logger().Errorf("Failed to get social account: %v", err)
		return &connector.PostResult{Err: fmt.Errorf("failed to get social account: %v", err), URL: ""}
	}

	req := BlueskyCreatePostRequest{
		Repo:       dbSocialAccount.Did,
		Collection: "app.bsky.feed.post",
		Record: BlueskyPostRecord{
			Type:      "app.bsky.feed.post",
			Text:      content,
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		},
	}

	res := BlueskyCreatePostResponse{}

	// Get the PDS endpoint for this social account
	pdsURL, pdsErr := b.getPDSEndpoint(dbSocialAccount)
	if pdsErr != nil {
		b.Logger().Errorf("Failed to get PDS endpoint: %v", pdsErr)
		return &connector.PostResult{Err: fmt.Errorf("failed to get PDS endpoint: %v", pdsErr), URL: ""}
	}

	b.Logger().Infof("Posting to Bluesky PDS at: %s", pdsURL)

	// Always use DPoP client for PDS calls (OAuth/DPoP tokens are valid only at the PDS)
	client := b.tmpClient
	if client == nil {
		client = b.ensureDPoPClient(dbSocialAccount)
		b.tmpClient = client
	}
	if client == nil {
		b.Logger().Errorf("Bluesky HTTP client not initialized; OAuth2 flow not completed")
		return &connector.PostResult{Err: fmt.Errorf("bluesky http client not initialized"), URL: ""}
	}

	b.Logger().Infof("Posting to Bluesky with request: %+v %+v", b, req)

	endpoint := pdsURL + "/xrpc/com.atproto.repo.createRecord"
	client.PostWithJson(endpoint, &req)
	client.AsJson(&res)

	b.Logger().Infof("Response from Bluesky: %+v", res)

	// Check for token-related errors in the response
	if client.Err != nil {
		if strings.Contains(client.Err.Error(), "invalid or expired OAuth token") {
			b.Logger().Warnf("Detected invalid token during post, attempting refresh")

			if refreshErr := b.refreshToken(dbSocialAccount); refreshErr != nil {
				b.Logger().Errorf("Failed to refresh token after detecting invalid token during post: %v", refreshErr)
				return &connector.PostResult{Err: fmt.Errorf("token refresh failed: %v", refreshErr), URL: ""}
			}
			// Reload the social account and retry the post
			if updatedAccount, err := b.db.GetSocialAccount(socialAccount.Id); err == nil {
				// Recreate the DPoP client with the new token
				if newClient := b.ensureDPoPClient(updatedAccount); newClient != nil {
					b.tmpClient = newClient
					client = newClient
					// Retry the post
					client.PostWithJson(endpoint, &req)
					client.AsJson(&res)
					b.Logger().Infof("Retry response from Bluesky: %+v", res)
				}
			} else {
				b.Logger().Errorf("Failed to get social account for token refresh: %v", err)
				return &connector.PostResult{Err: fmt.Errorf("failed to get social account: %v", err), URL: ""}
			}
		}

		// If we still have an error after potential retry, return it
		if client.Err != nil {
			b.Logger().Errorf("Error posting to Bluesky: %v", client.Err)
			return &connector.PostResult{Err: client.Err, URL: ""}
		}
	}

	// Build a web URL from the at:// URI: at://<repo>/app.bsky.feed.post/<rkey>
	rkey := res.Uri
	if idx := strings.LastIndex(rkey, "/"); idx >= 0 && idx+1 < len(rkey) {
		rkey = rkey[idx+1:]
	}

	// Use identity for URL, fallback to DID if identity is empty
	profileHandle := socialAccount.Identity
	if profileHandle == "" {
		profileHandle = socialAccount.Did
		b.Logger().Warnf("Using DID as fallback for profile URL since identity is empty: %s", profileHandle)
	}

	ret := &connector.PostResult{Err: nil, URL: "https://bsky.app/profile/" + profileHandle + "/post/" + rkey}

	return ret
}

func (b *BlueskyConnector) GetIcon() string {
	return "bi:bluesky"
}

func (b *BlueskyConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	user := socialAccount.Identity

	// Check if token is expired
	if socialAccount.OAuth2TokenExpiry.Before(time.Now()) {
		b.Logger().Warnf("OAuth token is expired, attempting to refresh")
		if err := b.refreshToken(socialAccount); err != nil {
			b.Logger().Errorf("Failed to refresh expired token: %v", err)
			return fmt.Errorf("token refresh failed: %v", err)
		}
		// Reload the social account to get updated token
		if updatedAccount, err := b.db.GetSocialAccount(socialAccount.ID); err == nil {
			socialAccount = updatedAccount
		}
	}

	// If identity is empty, try to get it from the token's sub claim
	if user == "" {
		b.Logger().Warnf("Social account identity is empty, attempting to get DID from token")

		// Extract identity (user DID) from token's 'sub' claim
		if socialAccount.OAuth2Token != "" {
			if did, err := b.extractDIDFromToken(socialAccount.OAuth2Token); err == nil {
				user = did
				b.Logger().Infof("Extracted user DID from token: %s", user)

				// Update the social account with the DID
				// TODO: Implement UpdateSocialAccountDID method in DB
				b.Logger().Warnf("Would update social account DID to: %s (method not implemented)", did)
			} else {
				b.Logger().Errorf("Failed to extract DID from token: %v", err)
				return fmt.Errorf("failed to extract DID from token: %v", err)
			}
		} else {
			b.Logger().Errorf("No OAuth token available to extract DID from")
			return fmt.Errorf("no OAuth token available")
		}
	}

	// If we have a DID but no handle, try to resolve the handle
	if user != "" && strings.HasPrefix(user, "did:") {
		b.Logger().Infof("Have DID but no handle, attempting to resolve handle for DID: %s", user)

		// Get handle from PLC directory or AppView
		if handle, err := b.getHandleFromDID(user); err == nil && handle != "" {
			b.Logger().Infof("Retrieved handle for DID %s: %s", user, handle)

			// Update the social account with the retrieved handle
			if err := b.db.UpdateSocialAccountIdentity(socialAccount.ID, handle); err != nil {
				b.Logger().Errorf("Failed to update social account identity: %v", err)
			}
		} else {
			b.Logger().Warnf("Failed to get handle for DID %s: %v", user, err)
			// Continue without handle - it may be available later
		}
	}

	// If we still don't have a user, skip the refresh
	if user == "" {
		b.Logger().Warnf("Skipping refresh: no user identity available")
		return nil
	}

	// If user is a DID, we don't need to resolve it further
	if strings.HasPrefix(user, "did:") {
		b.Logger().Debugf("User is already a DID, no further resolution needed: %s", user)
		return nil
	}

	// If user is a handle, try to resolve it (though this shouldn't happen with our new logic)
	b.Logger().Warnf("Unexpected handle in OnRefresh: %s", user)
	return nil
}

// refreshToken attempts to refresh an expired OAuth token with DPoP and JWT client assertion
func (b *BlueskyConnector) refreshToken(socialAccount *db.SocialAccount) error {
	if socialAccount.OAuth2RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Compose client_id from BASE_URL + /oauth/client-metadata.json
	baseURL := b.db.GetCvarString(db.CvarKeys.BaseUrl)
	clientId := baseURL + "/oauth/client-metadata.json"
	if clientId == "" {
		return fmt.Errorf("Bluesky BASE_URL not configured")
	}

	b.Logger().Infof("Using client_id for token refresh: %s", clientId)

	b.Logger().Infof("Refreshing OAuth token for Bluesky account ID: %d", socialAccount.ID)

	// Create a DPoP-enabled client for token refresh
	client := b.createDPoPClientForRefresh(socialAccount)
	if client == nil {
		return fmt.Errorf("cannot create DPoP client for token refresh")
	}

	// Create JWT client assertion
	clientAssertion, err := b.createClientAssertion(clientId)
	if err != nil {
		b.Logger().Errorf("Failed to create client assertion: %v", err)
		return fmt.Errorf("failed to create client assertion: %v", err)
	}

	// Prepare refresh token request with JWT client assertion
	refreshData := map[string]string{
		"grant_type":            "refresh_token",
		"refresh_token":         socialAccount.OAuth2RefreshToken,
		"client_id":             clientId,
		"client_assertion_type": "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
		"client_assertion":      clientAssertion,
	}

	// Make the refresh request to Bluesky's token endpoint
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}

	client.PostWithFormVars("https://bsky.social/oauth/token", refreshData).AsJson(&tokenResponse)

	if client.Err != nil {
		b.Logger().Errorf("Failed to refresh token: %v", client.Err)
		return fmt.Errorf("token refresh failed: %v", client.Err)
	}

	if tokenResponse.AccessToken == "" {
		b.Logger().Errorf("Received empty access token in refresh response")
		return fmt.Errorf("received empty access token")
	}

	// Calculate expiry time
	expiryTime := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)

	b.Logger().Infof("Token refreshed successfully, expires at: %v", expiryTime)

	// Update the social account with the new token
	err = b.db.UpdateSocialAccountToken(socialAccount.ID, tokenResponse.AccessToken, tokenResponse.RefreshToken, tokenResponse.ExpiresIn)
	if err != nil {
		b.Logger().Errorf("Failed to update social account token: %v", err)
		return fmt.Errorf("failed to update token in database: %v", err)
	}

	// Update the local social account object
	socialAccount.OAuth2Token = tokenResponse.AccessToken
	socialAccount.OAuth2RefreshToken = tokenResponse.RefreshToken
	socialAccount.OAuth2TokenExpiry = expiryTime

	// Recreate the DPoP client with the new token
	if newClient := b.ensureDPoPClient(socialAccount); newClient != nil {
		b.tmpClient = newClient
	}

	return nil
}

// createDPoPClientForRefresh creates a DPoP client specifically for token refresh operations
func (b *BlueskyConnector) createDPoPClientForRefresh(sa *db.SocialAccount) *utils.ChainingHttpClient {
	if sa == nil || !sa.DpopKey.Valid {
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

	// For token refresh, we don't include access token hash in DPoP proof
	// The ath claim is not allowed during token refresh operations
	tp := &dpopTransport{
		base:              client.UnderlyingClient().Transport,
		privKey:           priv,
		signedAccessToken: "", // No access token for refresh operations
		accessTokenHash:   "", // No ath claim for refresh operations
	}
	client.UnderlyingClient().Transport = tp
	return client
}

// resolveHandleToDID resolves a handle to a DID
func (b *BlueskyConnector) resolveHandleToDID(handle string) (string, error) {
	// Use public AppView for handle resolution (no auth required)
	url := fmt.Sprintf("https://api.bsky.app/xrpc/com.atproto.identity.resolveHandle?handle=%s", handle)

	client := utils.NewClient(b.Logger())
	var response struct {
		Did string `json:"did"`
	}

	client.Get(url).AsJson(&response)
	if client.Err != nil {
		return "", fmt.Errorf("failed to resolve handle %s: %v", handle, client.Err)
	}

	if response.Did == "" {
		return "", fmt.Errorf("no DID found for handle %s", handle)
	}

	return response.Did, nil
}

// resolveDIDToPDS resolves a DID to a PDS URL by resolving the DID document
func (b *BlueskyConnector) resolveDIDToPDS(did string) (string, error) {
	b.Logger().Infof("Resolving DID to PDS: %s", did)

	// Resolve the DID document
	didDoc, err := b.resolveDIDDocument(did)
	if err != nil {
		return "", fmt.Errorf("failed to resolve DID document for %s: %v", did, err)
	}

	// Extract PDS service endpoint from the DID document
	pdsURL, err := b.extractPDSServiceEndpoint(didDoc)
	if err != nil {
		return "", fmt.Errorf("failed to extract PDS service endpoint from DID document: %v", err)
	}

	b.Logger().Infof("Found PDS endpoint for DID %s: %s", did, pdsURL)
	return pdsURL, nil
}

// resolveDIDDocument resolves a DID to its document
func (b *BlueskyConnector) resolveDIDDocument(did string) (map[string]interface{}, error) {
	// Extract the DID method and identifier
	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid DID format: %s", did)
	}

	method := parts[1]
	identifier := parts[2]

	b.Logger().Debugf("Resolving DID document for method: %s, identifier: %s", method, identifier)

	// Handle different DID methods
	switch method {
	case "plc":
		return b.resolvePLCDIDDocument(identifier)
	case "web":
		return b.resolveWebDIDDocument(identifier)
	case "did":
		// Handle did:did:... format (some implementations use this)
		if len(parts) >= 4 {
			return b.resolveDIDDocument(strings.Join(parts[1:], ":"))
		}
		return nil, fmt.Errorf("unsupported DID method: %s", method)
	default:
		return nil, fmt.Errorf("unsupported DID method: %s", method)
	}
}

// resolvePLCDIDDocument resolves a PLC (Placeholder) DID document
func (b *BlueskyConnector) resolvePLCDIDDocument(identifier string) (map[string]interface{}, error) {
	// PLC DIDs are resolved via the PLC directory
	url := fmt.Sprintf("https://plc.directory/did:plc:%s", identifier)

	client := utils.NewClient(b.Logger())
	var didDoc map[string]interface{}

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		// Check if it's a 404 (DID not registered)
		if strings.Contains(client.Err.Error(), "404") || strings.Contains(client.Err.Error(), "Not Found") {
			return nil, fmt.Errorf("DID not registered: %s", identifier)
		}
		return nil, fmt.Errorf("failed to resolve PLC DID document: %v", client.Err)
	}

	return didDoc, nil
}

// resolveWebDIDDocument resolves a Web DID document
func (b *BlueskyConnector) resolveWebDIDDocument(identifier string) (map[string]interface{}, error) {
	// Web DIDs are resolved via the domain's well-known endpoint
	// Format: did:web:example.com -> https://example.com/.well-known/did.json
	url := fmt.Sprintf("https://%s/.well-known/did.json", identifier)

	client := utils.NewClient(b.Logger())
	var didDoc map[string]interface{}

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		return nil, fmt.Errorf("failed to resolve Web DID document: %v", client.Err)
	}

	return didDoc, nil
}

// extractPDSServiceEndpoint extracts the PDS service endpoint from a DID document
func (b *BlueskyConnector) extractPDSServiceEndpoint(didDoc map[string]interface{}) (string, error) {
	// Look for services in the DID document
	services, ok := didDoc["service"].([]interface{})
	if !ok {
		return "", fmt.Errorf("no services found in DID document")
	}

	// Find the AtprotoPersonalDataServer service
	for _, service := range services {
		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			continue
		}

		serviceType, ok := serviceMap["type"].(string)
		if !ok {
			continue
		}

		// Check if this is an AtprotoPersonalDataServer service
		if serviceType == "AtprotoPersonalDataServer" {
			serviceEndpoint, ok := serviceMap["serviceEndpoint"].(string)
			if !ok {
				continue
			}

			b.Logger().Debugf("Found AtprotoPersonalDataServer service: %s", serviceEndpoint)
			return serviceEndpoint, nil
		}
	}

	// If no AtprotoPersonalDataServer service found, try to find any atproto service
	for _, service := range services {
		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			continue
		}

		serviceType, ok := serviceMap["type"].(string)
		if !ok {
			continue
		}

		// Check if this is any atproto-related service
		if strings.Contains(strings.ToLower(serviceType), "atproto") ||
			strings.Contains(strings.ToLower(serviceType), "pds") ||
			strings.Contains(strings.ToLower(serviceType), "bsky") {
			serviceEndpoint, ok := serviceMap["serviceEndpoint"].(string)
			if !ok {
				continue
			}

			b.Logger().Debugf("Found atproto-related service: %s -> %s", serviceType, serviceEndpoint)
			return serviceEndpoint, nil
		}
	}

	return "", fmt.Errorf("no AtprotoPersonalDataServer service found in DID document")
}

// getPDSEndpoint gets the PDS endpoint for a given social account
func (b *BlueskyConnector) getPDSEndpoint(socialAccount *db.SocialAccount) (string, error) {
	// If we already have a DID, use it
	if socialAccount.Did != "" {
		return b.resolveDIDToPDS(socialAccount.Did)
	}

	// If we have an identity, check if it's a DID or handle
	if socialAccount.Identity != "" {
		// Check if identity looks like a DID
		if strings.HasPrefix(socialAccount.Identity, "did:") {
			// Identity is already a DID
			b.Logger().Infof("Identity is already a DID: %s", socialAccount.Identity)
			return b.resolveDIDToPDS(socialAccount.Identity)
		} else {
			// Identity is a handle, resolve it to DID first
			did, err := b.resolveHandleToDID(socialAccount.Identity)
			if err != nil {
				return "", fmt.Errorf("failed to resolve handle to DID: %v", err)
			}

			// Update the social account with the DID
			// TODO: Implement UpdateSocialAccountDID method in DB
			b.Logger().Warnf("Would update social account DID to: %s (method not implemented)", did)

			return b.resolveDIDToPDS(did)
		}
	}

	return "", fmt.Errorf("no handle or DID available for social account")
}

// createClientAssertion creates a JWT client assertion for OAuth2 client authentication
func (b *BlueskyConnector) createClientAssertion(clientId string) (string, error) {
	// Generate a new private key for client assertion
	// In a production system, you might want to store this key securely
	privKey, err := generatePrivateKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate client assertion key: %v", err)
	}

	now := time.Now()

	// JWT header
	header := map[string]interface{}{
		"typ": "JWT",
		"alg": "ES256",
		"jwk": map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(privKey.PublicKey.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(privKey.PublicKey.Y.Bytes()),
		},
	}

	// JWT payload
	payload := map[string]interface{}{
		"iss": clientId,                          // issuer (client_id)
		"sub": clientId,                          // subject (client_id)
		"aud": "https://bsky.social/oauth/token", // audience
		"iat": now.Unix(),                        // issued at
		"exp": now.Add(5 * time.Minute).Unix(),   // expires in 5 minutes
		"jti": uuid.NewString(),                  // JWT ID
	}

	// Encode header and payload
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT header: %v", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	dataToSign := encodedHeader + "." + encodedPayload

	// Sign the JWT
	hash := sha256.Sum256([]byte(dataToSign))
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %v", err)
	}

	// Concatenate r and s into a 64-byte signature
	sigBytes := make([]byte, 64)
	r.FillBytes(sigBytes[:32])
	s.FillBytes(sigBytes[32:])

	encodedSig := base64.RawURLEncoding.EncodeToString(sigBytes)
	jwt := dataToSign + "." + encodedSig

	b.Logger().Infof("Created JWT client assertion for client_id: %s", clientId)
	return jwt, nil
}

func (b *BlueskyConnector) GetOAuth2Config() *oauth2.Config {
	ep := oauth2.Endpoint{
		AuthURL:  "https://bsky.social/oauth/authorize",
		TokenURL: "https://bsky.social/oauth/token",
	}

	// Compose client_id from BASE_URL + /oauth/client-metadata.json
	baseURL := b.db.GetCvarString(db.CvarKeys.BaseUrl)
	clientID := baseURL + "/oauth/client-metadata.json"

	b.Logger().Infof("Using Bluesky OAuth2 client_id: %s", clientID)

	return &oauth2.Config{
		Endpoint:     ep,
		ClientID:     clientID,
		ClientSecret: "",
		Scopes:       []string{"atproto", "repo:app.bsky.feed.post?action=create"},
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
	accessTokenHash   string
	privKey           *ecdsa.PrivateKey
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Calculate access token hash if not already calculated
	if t.accessTokenHash == "" && t.signedAccessToken != "" {
		hash := sha256.Sum256([]byte(t.signedAccessToken))
		t.accessTokenHash = base64.RawURLEncoding.EncodeToString(hash[:])
	}

	dpopJwt, _ := createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce, t.accessTokenHash)

	req.Header.Set("DPoP", dpopJwt)
	if t.signedAccessToken != "" {
		req.Header.Set("Authorization", "DPoP "+t.signedAccessToken)
	}
	resp, err := t.base.RoundTrip(req)

	// Handle DPoP nonce retry and token errors
	if resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest) {
		// Read response body to check for specific errors
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(bodyBytes)

		// Check if this is a DPoP nonce error
		if strings.Contains(bodyStr, "use_dpop_nonce") {
			// Get nonce from response header
			nonce := resp.Header.Get("DPoP-Nonce")
			if nonce == "" {
				nonce = resp.Header.Get("Dpop-Nonce")
			}

			if nonce != "" {
				// Update nonce and retry once
				t.dpopNonce = nonce
				// Calculate access token hash if not already calculated
				if t.accessTokenHash == "" && t.signedAccessToken != "" {
					hash := sha256.Sum256([]byte(t.signedAccessToken))
					t.accessTokenHash = base64.RawURLEncoding.EncodeToString(hash[:])
				}
				dpopJwt, _ = createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce, t.accessTokenHash)

				// Clone request and retry
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

				resp2, err2 := t.base.RoundTrip(req2)
				if err2 == nil && resp2 != nil {
					return resp2, nil
				}
			}
		}

		// Check if this is a token-related error
		if strings.Contains(bodyStr, "InvalidToken") ||
			strings.Contains(bodyStr, "OAuth tokens are meant for PDS access only") {
			// This is a token error, not a DPoP nonce issue
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			return resp, fmt.Errorf("invalid or expired OAuth token")
		}

		// Restore body for other errors
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
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

	b.Logger().Debugf("Received token on exchange (token type: %s, expires in: %v)", token.TokenType, token.Expiry.Sub(time.Now()))

	// Extract identity (user DID) from token's 'sub' claim
	// The token contains the user's DID in the 'sub' field
	identity := ""
	if token.AccessToken != "" {
		// Parse JWT to extract 'sub' claim (user DID)
		if did, err := b.extractDIDFromToken(token.AccessToken); err == nil {
			identity = did
			b.Logger().Infof("Extracted user DID from token: %s", identity)
		} else {
			b.Logger().Warnf("Failed to extract DID from token: %v", err)
		}
	}

	// Resolve DID to PDS and handle
	handle := ""
	if identity != "" {
		// Get handle from PLC directory or AppView
		if h, err := b.getHandleFromDID(identity); err == nil && h != "" {
			handle = h
			b.Logger().Infof("Retrieved handle for DID %s: %s", identity, handle)
		} else {
			b.Logger().Warnf("Failed to get handle for DID %s: %v", identity, err)
		}
	}

	err = b.db.RegisterAccount(&db.SocialAccount{
		Connector:          "bluesky",
		Identity:           handle,
		Did:                identity,
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

	// Calculate access token hash
	var accessTokenHash string
	if sa.OAuth2Token != "" {
		hash := sha256.Sum256([]byte(sa.OAuth2Token))
		accessTokenHash = base64.RawURLEncoding.EncodeToString(hash[:])
	}

	tp := &dpopTransport{
		base:              client.UnderlyingClient().Transport,
		privKey:           priv,
		signedAccessToken: sa.OAuth2Token,
		accessTokenHash:   accessTokenHash,
	}
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
		b.Logger().Errorf("Failed to get session: %v", client.Err)
		return "", fmt.Errorf("session request failed: %v", client.Err)
	}
	return sr.Handle, nil
}

type resolveHandleResponse struct {
	Did string `json:"did"`
}

// extractDIDFromToken extracts the user DID from the JWT token's 'sub' claim
func (b *BlueskyConnector) extractDIDFromToken(accessToken string) (string, error) {
	// Parse JWT token to extract 'sub' claim
	// JWT format: header.payload.signature
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	// Decode payload (base64url)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	// Parse JSON payload
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload: %v", err)
	}

	// Extract 'sub' claim (user DID)
	sub, ok := payload["sub"].(string)
	if !ok {
		return "", fmt.Errorf("no 'sub' claim found in token")
	}

	return sub, nil
}

// getHandleFromDID gets the handle for a DID from PLC directory or AppView
func (b *BlueskyConnector) getHandleFromDID(did string) (string, error) {
	// First try to get handle from PLC directory (alsoKnownAs)
	if handle, err := b.getHandleFromPLC(did); err == nil && handle != "" {
		return handle, nil
	}

	// Fallback to AppView
	if handle, err := b.getHandleFromAppView(did); err == nil && handle != "" {
		return handle, nil
	}

	// If both fail, return empty handle (we have the DID which is sufficient)
	b.Logger().Warnf("Could not resolve handle for DID %s, using DID as identifier", did)
	return "", fmt.Errorf("handle resolution failed")
}

// getHandleFromPLC gets handle from PLC directory's alsoKnownAs field
func (b *BlueskyConnector) getHandleFromPLC(did string) (string, error) {
	// PLC directory expects the full DID including the did:plc: prefix
	url := fmt.Sprintf("https://plc.directory/%s", did)

	client := utils.NewClient(b.Logger())
	var didDoc map[string]interface{}

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		return "", fmt.Errorf("failed to get PLC document: %v", client.Err)
	}

	// Look for alsoKnownAs field
	alsoKnownAs, ok := didDoc["alsoKnownAs"].([]interface{})
	if !ok || len(alsoKnownAs) == 0 {
		return "", fmt.Errorf("no alsoKnownAs found in PLC document")
	}

	// Get first alsoKnownAs entry
	firstEntry, ok := alsoKnownAs[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid alsoKnownAs format")
	}

	// Extract handle from at://alice.bsky.social format
	if strings.HasPrefix(firstEntry, "at://") {
		handle := strings.TrimPrefix(firstEntry, "at://")
		return handle, nil
	}

	return "", fmt.Errorf("unexpected alsoKnownAs format: %s", firstEntry)
}

// getHandleFromAppView gets handle from AppView using DID
func (b *BlueskyConnector) getHandleFromAppView(did string) (string, error) {
	// Use public AppView (api.bsky.app) which doesn't require auth for public profiles
	url := fmt.Sprintf("https://api.bsky.app/xrpc/app.bsky.actor.getProfile?actor=%s", did)

	client := utils.NewClient(b.Logger())
	var response struct {
		Handle string `json:"handle"`
	}

	client.Get(url).AsJson(&response)
	if client.Err != nil {
		// Check if it's a 401 (authentication required for this DID)
		if strings.Contains(client.Err.Error(), "401") || strings.Contains(client.Err.Error(), "AuthMissing") {
			b.Logger().Warnf("Profile for DID %s requires authentication, skipping handle resolution", did)
			return "", fmt.Errorf("profile requires authentication")
		}
		return "", fmt.Errorf("failed to get profile from AppView: %v", client.Err)
	}

	return response.Handle, nil
}
