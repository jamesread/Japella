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
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	Type      string         `json:"$type"`
	Text      string         `json:"text"`
	CreatedAt string         `json:"createdAt"`
	Facets    []BlueskyFacet `json:"facets,omitempty"`
	Embed     any            `json:"embed,omitempty"`
	// Reply makes the post a reply; Root is the thread's first post.
	Reply *BlueskyReplyRef `json:"reply,omitempty"`
}

// BlueskyStrongRef points at an exact version of a record.
type BlueskyStrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// BlueskyReplyRef is app.bsky.feed.post#replyRef.
type BlueskyReplyRef struct {
	Root   BlueskyStrongRef `json:"root"`
	Parent BlueskyStrongRef `json:"parent"`
}

// BlueskyEmbedImages is app.bsky.embed.images for image posts
type BlueskyEmbedImages struct {
	Type   string                  `json:"$type"`
	Images []BlueskyEmbedImageItem `json:"images"`
}

// BlueskyEmbedExternal is app.bsky.embed.external for rich link cards
type BlueskyEmbedExternal struct {
	Type     string          `json:"$type"`
	External BlueskyExternal `json:"external"`
}

// BlueskyExternal is app.bsky.embed.external#external (uri, title, description, optional thumb)
type BlueskyExternal struct {
	URI         string          `json:"uri"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Thumb       *BlueskyBlobRef `json:"thumb,omitempty"`
}

type BlueskyEmbedImageItem struct {
	Image       BlueskyBlobRef      `json:"image"`
	Alt         string              `json:"alt"`
	AspectRatio *BlueskyAspectRatio `json:"aspectRatio,omitempty"`
}

type BlueskyBlobRef struct {
	Type     string          `json:"$type"`
	Ref      BlueskyBlobLink `json:"ref"`
	MimeType string          `json:"mimeType"`
	Size     int64           `json:"size"`
}

type BlueskyBlobLink struct {
	Link string `json:"$link"`
}

type BlueskyAspectRatio struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// uploadBlobResponse is the response from POST com.atproto.repo.uploadBlob
type uploadBlobResponse struct {
	Blob BlueskyBlobRef `json:"blob"`
}

type BlueskyFacet struct {
	Index    BlueskyFacetIndex     `json:"index"`
	Features []BlueskyFacetFeature `json:"features"`
}

type BlueskyFacetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type BlueskyFacetFeature struct {
	Type string `json:"$type"`
	// For links
	URI string `json:"uri,omitempty"`
	// For mentions
	DID string `json:"did,omitempty"`
	// For hashtags
	Tag string `json:"tag,omitempty"`
}

type BlueskyCreatePostResponse struct {
	Uri string `json:"uri"`
	Cid string `json:"cid"`
}

func blobRefMimeToContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func imageAspectRatio(path string) (w, h int) {
	w, h = 1, 1
	f, err := os.Open(path) //nolint:gosec // media path resolved by media.PathFromURL
	if err != nil {
		return 1, 1
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 1, 1
	}
	if cfg.Width < 1 {
		cfg.Width = 1
	}
	if cfg.Height < 1 {
		cfg.Height = 1
	}
	return cfg.Width, cfg.Height
}

func (b *BlueskyConnector) uploadBlob(pdsURL, path string, client *utils.ChainingHttpClient) (*BlueskyBlobRef, error) {
	data, err := os.ReadFile(path) //nolint:gosec // media path resolved by media.PathFromURL
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	ext := filepath.Ext(path)
	contentType := blobRefMimeToContentType(ext)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, pdsURL+"/xrpc/com.atproto.repo.uploadBlob", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	resp, err := client.UnderlyingClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("uploadBlob returned %d: %s", resp.StatusCode, string(body))
	}

	var out uploadBlobResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if out.Blob.Ref.Link == "" {
		return nil, fmt.Errorf("uploadBlob response missing blob.ref: %s", string(body))
	}
	out.Blob.Type = "blob"
	return &out.Blob, nil
}

const (
	thumbMaxSize = 1000000 // 1MB Bluesky limit for external thumb
	thumbTimeout = 15 * time.Second
)

// downloadAndUploadThumb fetches an image from imageURL, saves to a temp file, uploads as blob to PDS, and returns the blob ref.
func (b *BlueskyConnector) downloadAndUploadThumb(pdsURL, imageURL string, client *utils.ChainingHttpClient) *BlueskyBlobRef {
	httpClient := &http.Client{Timeout: thumbTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, imageURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", ogUserAgent)
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, thumbMaxSize+1))
	if err != nil || len(body) > thumbMaxSize {
		return nil
	}
	ext := ".jpg"
	switch {
	case strings.HasPrefix(ct, "image/png"):
		ext = ".png"
	case strings.HasPrefix(ct, "image/gif"):
		ext = ".gif"
	case strings.HasPrefix(ct, "image/webp"):
		ext = ".webp"
	}
	tmp, err := os.CreateTemp("", "japella-thumb-*"+ext)
	if err != nil {
		return nil
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err = tmp.Write(body); err != nil {
		_ = tmp.Close()
		return nil
	}
	if err = tmp.Close(); err != nil {
		return nil
	}
	blob, err := b.uploadBlob(pdsURL, tmpPath, client)
	if err != nil {
		return nil
	}
	return blob
}

func (b *BlueskyConnector) PostToWall(socialAccount *connector.SocialAccount, content string, mediaPaths []string) *connector.PostResult {
	// The full account holds the OAuth fields.
	dbSocialAccount, err := b.db.GetSocialAccount(socialAccount.Id)
	if err != nil {
		b.Logger().Errorf("Failed to get social account: %v", err)
		return &connector.PostResult{Err: fmt.Errorf("failed to get social account: %w", err), URL: ""}
	}

	pdsURL, client, err := b.postingTarget(dbSocialAccount)
	if err != nil {
		return &connector.PostResult{Err: err, URL: ""}
	}

	record, err := b.buildPostRecord(pdsURL, client, content, mediaPaths)
	if err != nil {
		b.db.InsertTableLog(fmt.Sprintf("Bluesky blob upload failed for account %d: %v", socialAccount.Id, err), "error", &socialAccount.Id)
		return &connector.PostResult{Err: err, URL: ""}
	}

	req := &BlueskyCreatePostRequest{Repo: dbSocialAccount.Did, Collection: "app.bsky.feed.post", Record: record}
	res, err := b.publishRecord(client, pdsURL+"/xrpc/com.atproto.repo.createRecord", req, dbSocialAccount)
	if err != nil {
		return &connector.PostResult{Err: err, URL: ""}
	}
	return &connector.PostResult{Err: nil, URL: b.postWebURL(socialAccount, res.Uri)}
}

// postingTarget returns the account's PDS and the DPoP client for it;
// OAuth/DPoP tokens are only valid at the PDS.
func (b *BlueskyConnector) postingTarget(sa *db.SocialAccount) (string, *utils.ChainingHttpClient, error) {
	pdsURL, err := b.getPDSEndpoint(sa)
	if err != nil {
		b.Logger().Errorf("Failed to get PDS endpoint: %v", err)
		return "", nil, fmt.Errorf("failed to get PDS endpoint: %w", err)
	}
	b.Logger().Infof("Posting to Bluesky PDS at: %s", pdsURL)

	if b.tmpClient == nil {
		b.tmpClient = b.ensureDPoPClient(sa)
	}
	if b.tmpClient == nil {
		b.Logger().Errorf("Bluesky HTTP client not initialized; OAuth2 flow not completed")
		return "", nil, fmt.Errorf("bluesky http client not initialized")
	}
	return pdsURL, b.tmpClient, nil
}

func (b *BlueskyConnector) buildPostRecord(pdsURL string, client *utils.ChainingHttpClient, content string, mediaPaths []string) (BlueskyPostRecord, error) {
	record := BlueskyPostRecord{
		Type:      "app.bsky.feed.post",
		Text:      content,
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Facets:    b.parseFacets(content),
	}
	embed, err := b.buildEmbed(pdsURL, client, content, mediaPaths)
	record.Embed = embed
	return record, err
}

// buildEmbed attaches images, or failing that a link card for the first URL.
// It returns an untyped nil when there is nothing to embed, so "embed" is omitted.
func (b *BlueskyConnector) buildEmbed(pdsURL string, client *utils.ChainingHttpClient, content string, mediaPaths []string) (any, error) {
	if len(mediaPaths) > 0 {
		return b.imagesEmbed(pdsURL, client, mediaPaths)
	}
	if card := b.linkCardEmbed(pdsURL, client, content); card != nil {
		return card, nil
	}
	return nil, nil
}

const maxBlueskyImages = 4

func (b *BlueskyConnector) imagesEmbed(pdsURL string, client *utils.ChainingHttpClient, mediaPaths []string) (*BlueskyEmbedImages, error) {
	if len(mediaPaths) > maxBlueskyImages {
		b.Logger().Warnf("Bluesky allows max 4 images per post, skipping remaining %d", len(mediaPaths)-maxBlueskyImages)
		mediaPaths = mediaPaths[:maxBlueskyImages]
	}
	images := make([]BlueskyEmbedImageItem, 0, len(mediaPaths))
	for _, p := range mediaPaths {
		item, err := b.uploadImage(pdsURL, client, p)
		if err != nil {
			return nil, err
		}
		images = append(images, item)
	}
	return &BlueskyEmbedImages{Type: "app.bsky.embed.images", Images: images}, nil
}

func (b *BlueskyConnector) uploadImage(pdsURL string, client *utils.ChainingHttpClient, path string) (BlueskyEmbedImageItem, error) {
	blob, err := b.uploadBlob(pdsURL, path, client)
	if err != nil {
		b.Logger().Errorf("Bluesky blob upload failed for %s: %v", path, err)
		return BlueskyEmbedImageItem{}, err
	}
	aw, ah := imageAspectRatio(path)
	return BlueskyEmbedImageItem{Image: *blob, Alt: "", AspectRatio: &BlueskyAspectRatio{Width: aw, Height: ah}}, nil
}

// linkCardEmbed builds an app.bsky.embed.external card from the first URL's
// Open Graph data, or returns nil.
func (b *BlueskyConnector) linkCardEmbed(pdsURL string, client *utils.ChainingHttpClient, content string) *BlueskyEmbedExternal {
	firstURL := b.getFirstURL(content)
	if firstURL == "" {
		return nil
	}
	preview, err := b.fetchLinkPreview(firstURL)
	if err != nil || preview == nil {
		return nil
	}
	ext := BlueskyExternal{URI: firstURL, Title: preview.Title, Description: preview.Description}
	if preview.ImageURL != "" {
		ext.Thumb = b.downloadAndUploadThumb(pdsURL, preview.ImageURL, client)
	}
	return &BlueskyEmbedExternal{Type: "app.bsky.embed.external", External: ext}
}

// publishRecord creates the post, refreshing the token and retrying once if
// Bluesky reports it expired.
func (b *BlueskyConnector) publishRecord(client *utils.ChainingHttpClient, endpoint string, req *BlueskyCreatePostRequest, sa *db.SocialAccount) (*BlueskyCreatePostResponse, error) {
	res := &BlueskyCreatePostResponse{}
	b.Logger().Infof("Posting to Bluesky with request: %+v %+v", b, *req)
	client.PostWithJson(endpoint, req)
	client.AsJson(res)
	b.Logger().Infof("Response from Bluesky: %+v", *res)
	if client.Err == nil {
		return res, nil
	}

	b.recordPostFailure(client, sa.ID)
	if strings.Contains(client.Err.Error(), "invalid or expired OAuth token") {
		retried, err := b.retryAfterTokenRefresh(client, endpoint, req, res, sa)
		if err != nil {
			return nil, err
		}
		client = retried
	}
	if client.Err != nil {
		b.logClientError("Error posting to Bluesky (after retry)", client)
		return nil, client.Err
	}
	return res, nil
}

// retryAfterTokenRefresh returns the client used for the retry, or the
// original client when no new one could be made.
func (b *BlueskyConnector) retryAfterTokenRefresh(client *utils.ChainingHttpClient, endpoint string, req *BlueskyCreatePostRequest, res *BlueskyCreatePostResponse, sa *db.SocialAccount) (*utils.ChainingHttpClient, error) {
	b.Logger().Warnf("Detected invalid token during post, attempting refresh")
	if err := b.refreshToken(sa); err != nil {
		b.Logger().Errorf("Failed to refresh token after detecting invalid token during post: %v", err)
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	updated, err := b.db.GetSocialAccount(sa.ID)
	if err != nil {
		b.Logger().Errorf("Failed to get social account for token refresh: %v", err)
		return nil, fmt.Errorf("failed to get social account: %w", err)
	}
	newClient := b.ensureDPoPClient(updated)
	if newClient == nil {
		return client, nil
	}
	b.tmpClient = newClient
	newClient.PostWithJson(endpoint, req)
	newClient.AsJson(res)
	b.Logger().Infof("Retry response from Bluesky: %+v", *res)
	return newClient, nil
}

func (b *BlueskyConnector) recordPostFailure(client *utils.ChainingHttpClient, accountID uint32) {
	b.logClientError("Error posting to Bluesky", client)
	msg := fmt.Sprintf("Error posting to Bluesky for account %d: %v", accountID, client.Err)
	if len(client.ResBody) > 0 {
		msg += " | response: " + string(client.ResBody)
	}
	b.db.InsertTableLog(msg, "error", &accountID)
}

func (b *BlueskyConnector) logClientError(label string, client *utils.ChainingHttpClient) {
	b.Logger().Errorf("%s: %v", label, client.Err)
	if len(client.ResBody) > 0 {
		b.Logger().Errorf("Bluesky API response body: %s", string(client.ResBody))
	}
}

// postWebURL builds the bsky.app URL from at://<repo>/app.bsky.feed.post/<rkey>.
func (b *BlueskyConnector) postWebURL(sa *connector.SocialAccount, uri string) string {
	rkey := uri
	if idx := strings.LastIndex(rkey, "/"); idx >= 0 && idx+1 < len(rkey) {
		rkey = rkey[idx+1:]
	}
	profileHandle := sa.Identity
	if profileHandle == "" {
		profileHandle = sa.Did
		b.Logger().Warnf("Using DID as fallback for profile URL since identity is empty: %s", profileHandle)
	}
	return "https://bsky.app/profile/" + profileHandle + "/post/" + rkey
}

// parseFacets parses URLs, mentions, and hashtags from text and returns Bluesky facets
// getFirstURL returns the first valid http(s) URL in text, or empty string.
func (b *BlueskyConnector) getFirstURL(text string) string {
	if text == "" {
		return ""
	}
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	m := urlRegex.FindString(text)
	if m == "" {
		return ""
	}
	if parsed, err := url.Parse(m); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return m
	}
	return ""
}

func (b *BlueskyConnector) parseFacets(text string) []BlueskyFacet {
	if text == "" {
		return nil
	}

	facets := []BlueskyFacet{}

	// Regular expressions for matching
	// URLs: http:// or https:// followed by non-whitespace characters
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	// Mentions: @ followed by a valid domain (e.g., @alice.bsky.social)
	// Bluesky handles are domain names, so we match @domain.tld format
	mentionRegex := regexp.MustCompile(`@([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?`)
	// Hashtags: # followed by letters, numbers, and underscores
	hashtagRegex := regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

	// Track used byte ranges to avoid overlaps
	usedRanges := make(map[int]bool)

	// Find URLs
	urlMatches := urlRegex.FindAllStringIndex(text, -1)
	for _, match := range urlMatches {
		start, end := match[0], match[1]

		// Check if this range overlaps with any existing facet
		overlaps := false
		for i := start; i < end; i++ {
			if usedRanges[i] {
				overlaps = true
				break
			}
		}

		if !overlaps {
			// Calculate byte offsets
			byteStart := len([]byte(text[:start]))
			byteEnd := len([]byte(text[:end]))

			// Extract URL
			urlStr := text[start:end]

			// Validate URL
			if parsedURL, err := url.Parse(urlStr); err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
				facet := BlueskyFacet{
					Index: BlueskyFacetIndex{
						ByteStart: byteStart,
						ByteEnd:   byteEnd,
					},
					Features: []BlueskyFacetFeature{
						{
							Type: "app.bsky.richtext.facet#link",
							URI:  urlStr,
						},
					},
				}
				facets = append(facets, facet)

				// Mark range as used
				for i := start; i < end; i++ {
					usedRanges[i] = true
				}
			}
		}
	}

	// Find mentions (@handle)
	mentionMatches := mentionRegex.FindAllStringIndex(text, -1)
	for _, match := range mentionMatches {
		start, end := match[0], match[1]

		// Check if this range overlaps with any existing facet
		overlaps := false
		for i := start; i < end; i++ {
			if usedRanges[i] {
				overlaps = true
				break
			}
		}

		if !overlaps {
			// Calculate byte offsets
			byteStart := len([]byte(text[:start]))
			byteEnd := len([]byte(text[:end]))

			// Extract handle (without @)
			handle := text[start+1 : end]

			// Try to resolve handle to DID
			did, err := b.resolveHandleToDID(handle)
			if err != nil {
				b.Logger().Warnf("Failed to resolve handle %s to DID: %v", handle, err)
				// Continue without DID - Bluesky may still accept the mention
				did = ""
			}

			feature := BlueskyFacetFeature{
				Type: "app.bsky.richtext.facet#mention",
			}
			if did != "" {
				feature.DID = did
			}

			facet := BlueskyFacet{
				Index: BlueskyFacetIndex{
					ByteStart: byteStart,
					ByteEnd:   byteEnd,
				},
				Features: []BlueskyFacetFeature{feature},
			}
			facets = append(facets, facet)

			// Mark range as used
			for i := start; i < end; i++ {
				usedRanges[i] = true
			}
		}
	}

	// Find hashtags
	hashtagMatches := hashtagRegex.FindAllStringIndex(text, -1)
	for _, match := range hashtagMatches {
		start, end := match[0], match[1]

		// Check if this range overlaps with any existing facet
		overlaps := false
		for i := start; i < end; i++ {
			if usedRanges[i] {
				overlaps = true
				break
			}
		}

		if !overlaps {
			// Calculate byte offsets
			byteStart := len([]byte(text[:start]))
			byteEnd := len([]byte(text[:end]))

			// Extract tag (without #)
			tag := text[start+1 : end]

			facet := BlueskyFacet{
				Index: BlueskyFacetIndex{
					ByteStart: byteStart,
					ByteEnd:   byteEnd,
				},
				Features: []BlueskyFacetFeature{
					{
						Type: "app.bsky.richtext.facet#tag",
						Tag:  tag,
					},
				},
			}
			facets = append(facets, facet)

			// Mark range as used
			for i := start; i < end; i++ {
				usedRanges[i] = true
			}
		}
	}

	return facets
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
			return fmt.Errorf("token refresh failed: %w", err)
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
				return fmt.Errorf("failed to extract DID from token: %w", err)
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
			if err = b.db.UpdateSocialAccountIdentity(socialAccount.ID, handle); err != nil {
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
		return fmt.Errorf("bluesky BASE_URL not configured")
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
		return fmt.Errorf("failed to create client assertion: %w", err)
	}

	// Prepare refresh token request with JWT client assertion
	refreshData := map[string]string{
		"grant_type":            "refresh_token",
		"refresh_token":         socialAccount.OAuth2RefreshToken,
		"client_id":             clientId,
		"client_assertion_type": "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
		"client_assertion":      clientAssertion,
	}

	// Build form data
	form := url.Values{}
	for key, value := range refreshData {
		form.Add(key, value)
	}

	// Create HTTP request manually so we can read error responses
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://bsky.social/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		b.Logger().Errorf("Failed to create refresh token request: %v", err)
		return fmt.Errorf("token refresh failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute the request using the DPoP client
	resp, err := client.UnderlyingClient().Do(req)
	if err != nil {
		b.Logger().Errorf("Failed to execute refresh token request: %v", err)
		return fmt.Errorf("token refresh failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response body
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		b.Logger().Errorf("Failed to read response body: %v", readErr)
		return fmt.Errorf("token refresh failed: failed to read response: %w", readErr)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := string(bodyBytes)
		b.Logger().Errorf("Token refresh failed with status %d. Response body: %s", resp.StatusCode, bodyStr)

		// Try to parse error response as JSON
		var errorResponse map[string]any
		if json.Unmarshal(bodyBytes, &errorResponse) == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				errorDesc := ""
				if desc, ok := errorResponse["error_description"].(string); ok {
					errorDesc = ": " + desc
				}

				// Check for invalid refresh token - this requires re-authentication
				if errorMsg == "invalid_grant" && strings.Contains(strings.ToLower(errorDesc), "refresh token") {
					b.Logger().Warnf("Refresh token is invalid or expired for account ID %d. User needs to re-authenticate.", socialAccount.ID)
					return fmt.Errorf("refresh token invalid or expired - re-authentication required%s", errorDesc)
				}

				return fmt.Errorf("token refresh failed: %s%s", errorMsg, errorDesc)
			}
		}
		return fmt.Errorf("token refresh failed: unexpected status code: %d", resp.StatusCode)
	}

	// Parse successful response
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err = json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
		b.Logger().Errorf("Failed to parse token response: %v", err)
		return fmt.Errorf("token refresh failed: failed to parse response: %w", err)
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
		return fmt.Errorf("failed to update token in database: %w", err)
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

	priv, err := decodeStoredDPoPKey(sa.DpopKey.String)
	if err != nil {
		b.Logger().Warnf("Cannot load DPoP key for account %d: %v", sa.ID, err)
		return nil
	}

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
		return "", fmt.Errorf("failed to resolve handle %s: %w", handle, client.Err)
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
		return "", fmt.Errorf("failed to resolve DID document for %s: %w", did, err)
	}

	// Extract PDS service endpoint from the DID document
	pdsURL, err := b.extractPDSServiceEndpoint(didDoc)
	if err != nil {
		return "", fmt.Errorf("failed to extract PDS service endpoint from DID document: %w", err)
	}

	b.Logger().Infof("Found PDS endpoint for DID %s: %s", did, pdsURL)
	return pdsURL, nil
}

// resolveDIDDocument resolves a DID to its document
func (b *BlueskyConnector) resolveDIDDocument(did string) (map[string]any, error) {
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
func (b *BlueskyConnector) resolvePLCDIDDocument(identifier string) (map[string]any, error) {
	// PLC DIDs are resolved via the PLC directory
	url := fmt.Sprintf("https://plc.directory/did:plc:%s", identifier)

	client := utils.NewClient(b.Logger())
	var didDoc map[string]any

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		// Check if it's a 404 (DID not registered)
		if strings.Contains(client.Err.Error(), "404") || strings.Contains(client.Err.Error(), "Not Found") {
			return nil, fmt.Errorf("DID not registered: %s", identifier)
		}
		return nil, fmt.Errorf("failed to resolve PLC DID document: %w", client.Err)
	}

	return didDoc, nil
}

// resolveWebDIDDocument resolves a Web DID document
func (b *BlueskyConnector) resolveWebDIDDocument(identifier string) (map[string]any, error) {
	// Web DIDs are resolved via the domain's well-known endpoint
	// Format: did:web:example.com -> https://example.com/.well-known/did.json
	url := fmt.Sprintf("https://%s/.well-known/did.json", identifier)

	client := utils.NewClient(b.Logger())
	var didDoc map[string]any

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		return nil, fmt.Errorf("failed to resolve Web DID document: %w", client.Err)
	}

	return didDoc, nil
}

// extractPDSServiceEndpoint extracts the PDS service endpoint from a DID document
func (b *BlueskyConnector) extractPDSServiceEndpoint(didDoc map[string]any) (string, error) {
	// Look for services in the DID document
	services, ok := didDoc["service"].([]any)
	if !ok {
		return "", fmt.Errorf("no services found in DID document")
	}

	// Find the AtprotoPersonalDataServer service
	for _, service := range services {
		serviceMap, ok := service.(map[string]any)
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
		serviceMap, ok := service.(map[string]any)
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
				return "", fmt.Errorf("failed to resolve handle to DID: %w", err)
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
		return "", fmt.Errorf("failed to generate client assertion key: %w", err)
	}

	now := time.Now()

	// JWT header
	header := map[string]any{
		"typ": "JWT",
		"alg": "ES256",
		"jwk": map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(privKey.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(privKey.Y.Bytes()),
		},
	}

	// JWT payload
	payload := map[string]any{
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
		return "", fmt.Errorf("failed to marshal JWT header: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	dataToSign := encodedHeader + "." + encodedPayload

	// Sign the JWT
	hash := sha256.Sum256([]byte(dataToSign))
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
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
	ep := oauth2.Endpoint{ //nolint:gosec // public OAuth endpoint URLs, not credentials
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
		Scopes:       []string{"atproto", "repo:app.bsky.feed.post?action=create", "blob:image/png", "blob:image/jpeg", "blob:image/gif", "blob:image/webp"},
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

func (b *BlueskyConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	res := &connector.ConfigurationCheckResult{
		Issues: []connector.ConfigurationIssue{},
	}

	baseURL := b.db.GetCvarString(db.CvarKeys.BaseUrl)
	if baseURL == "" || baseURL == "http://localhost:8080" {
		res.AddSettingsIssue("Base URL must be set to your public Japella URL for Bluesky OAuth.", db.CvarKeys.BaseUrl)
	}

	redirectURL := b.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL)
	expectedRedirect := baseURL + "/oauth2callback"
	if redirectURL == "" {
		res.AddSettingsIssue("OAuth2 Redirect URL is not set.", db.CvarKeys.OAuth2RedirectURL)
	} else if baseURL != "" && redirectURL != expectedRedirect {
		res.AddSettingsIssue("OAuth2 Redirect URL should match your base URL plus /oauth2callback.", db.CvarKeys.OAuth2RedirectURL)
	}

	if !b.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible) {
		res.AddSettingsIssue("Bluesky OAuth requires Is Publicly Accessible to be enabled.", db.CvarKeys.IsPubliclyAccessible)
	}

	return res
}

type dpopTransport struct {
	base              http.RoundTripper
	dpopNonce         string
	signedAccessToken string
	accessTokenHash   string
	privKey           *ecdsa.PrivateKey
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Calculate access token hash if not already calculated
	if t.accessTokenHash == "" && t.signedAccessToken != "" {
		t.accessTokenHash = calculateAccessTokenHash(t.signedAccessToken)
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
		_ = resp.Body.Close()
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
					t.accessTokenHash = calculateAccessTokenHash(t.signedAccessToken)
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

	b.Logger().Debugf("Received token on exchange (token type: %s, expires in: %v)", token.TokenType, time.Until(token.Expiry))

	// Extract identity (user DID) from token's 'sub' claim
	// The token contains the user's DID in the 'sub' field
	identity := ""
	if token.AccessToken != "" {
		// Parse JWT to extract 'sub' claim (user DID)
		if did, didErr := b.extractDIDFromToken(token.AccessToken); didErr == nil {
			identity = did
			b.Logger().Infof("Extracted user DID from token: %s", identity)
		} else {
			b.Logger().Warnf("Failed to extract DID from token: %v", didErr)
		}
	}

	// Resolve DID to PDS and handle
	handle := ""
	if identity != "" {
		// Get handle from PLC directory or AppView
		if h, handleErr := b.getHandleFromDID(identity); handleErr == nil && h != "" {
			handle = h
			b.Logger().Infof("Retrieved handle for DID %s: %s", identity, handle)
		} else {
			b.Logger().Warnf("Failed to get handle for DID %s: %v", identity, handleErr)
		}
	}

	storedKey, err := encodeStoredDPoPKey(privKey)
	if err != nil {
		return err
	}

	err = b.db.RegisterAccount(&db.SocialAccount{
		Connector:          "bluesky",
		Identity:           handle,
		Did:                identity,
		Homeserver:         "https://bsky.social",
		Active:             true,
		State:              "active",
		OAuth2Token:        token.AccessToken,
		OAuth2TokenExpiry:  token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
		DpopKey:            sql.NullString{String: storedKey, Valid: true},
	})

	return err
}

// ensureDPoPClient reconstructs a DPoP-enabled HTTP client from stored DB fields
func (b *BlueskyConnector) ensureDPoPClient(sa *db.SocialAccount) *utils.ChainingHttpClient {
	if sa == nil || !sa.DpopKey.Valid || sa.OAuth2Token == "" {
		return nil
	}
	priv, err := decodeStoredDPoPKey(sa.DpopKey.String)
	if err != nil {
		b.Logger().Warnf("Cannot load DPoP key for account %d: %v", sa.ID, err)
		return nil
	}

	client := utils.NewClient(b.Logger())

	// Calculate access token hash
	var accessTokenHash string
	if sa.OAuth2Token != "" {
		accessTokenHash = calculateAccessTokenHash(sa.OAuth2Token)
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
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse JSON payload
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload: %w", err)
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
	var didDoc map[string]any

	client.Get(url).AsJson(&didDoc)
	if client.Err != nil {
		return "", fmt.Errorf("failed to get PLC document: %w", client.Err)
	}

	// Look for alsoKnownAs field
	alsoKnownAs, ok := didDoc["alsoKnownAs"].([]any)
	if !ok || len(alsoKnownAs) == 0 {
		return "", fmt.Errorf("no alsoKnownAs found in PLC document")
	}

	// Get first alsoKnownAs entry
	firstEntry, ok := alsoKnownAs[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid alsoKnownAs format")
	}

	// Extract handle from at://alice.bsky.social format
	if handle, found := strings.CutPrefix(firstEntry, "at://"); found {
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
		return "", fmt.Errorf("failed to get profile from AppView: %w", client.Err)
	}

	return response.Handle, nil
}
