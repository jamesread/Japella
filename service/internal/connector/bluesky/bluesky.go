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
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
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
	req, err := newUploadBlobRequest(pdsURL, path, data)
	if err != nil {
		return nil, err
	}
	resp, err := client.UnderlyingClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeUploadBlobResponse(resp)
}

func newUploadBlobRequest(pdsURL, path string, data []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, pdsURL+"/xrpc/com.atproto.repo.uploadBlob", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", blobRefMimeToContentType(filepath.Ext(path)))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return req, nil
}

func decodeUploadBlobResponse(resp *http.Response) (*BlueskyBlobRef, error) {
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
	body, ext, ok := fetchThumbImage(imageURL)
	if !ok {
		return nil
	}
	tmpPath, err := writeThumbTempFile(body, ext)
	if err != nil {
		return nil
	}
	defer func() { _ = os.Remove(tmpPath) }()
	blob, err := b.uploadBlob(pdsURL, tmpPath, client)
	if err != nil {
		return nil
	}
	return blob
}

func fetchThumbImage(imageURL string) (body []byte, ext string, ok bool) {
	resp, err := doThumbGET(imageURL)
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = resp.Body.Close() }()
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, "", false
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, thumbMaxSize+1))
	if err != nil || len(body) > thumbMaxSize {
		return nil, "", false
	}
	return body, thumbExtForContentType(ct), true
}

func doThumbGET(imageURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ogUserAgent)
	resp, err := (&http.Client{Timeout: thumbTimeout}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("thumb fetch status %d", resp.StatusCode)
	}
	return resp, nil
}

func thumbExtForContentType(ct string) string {
	switch {
	case strings.HasPrefix(ct, "image/png"):
		return ".png"
	case strings.HasPrefix(ct, "image/gif"):
		return ".gif"
	case strings.HasPrefix(ct, "image/webp"):
		return ".webp"
	default:
		return ".jpg"
	}
}

func writeThumbTempFile(body []byte, ext string) (string, error) {
	tmp, err := os.CreateTemp("", "japella-thumb-*"+ext)
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err = tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
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

// getFirstURL returns the first valid http(s) URL in text, or empty string.
func (b *BlueskyConnector) getFirstURL(text string) string {
	if text == "" {
		return ""
	}
	m := facetURLRegex.FindString(text)
	if m == "" {
		return ""
	}
	if _, ok := linkFeature(m); ok {
		return m
	}
	return ""
}

var (
	// URLs: http:// or https:// followed by non-whitespace characters.
	facetURLRegex = regexp.MustCompile(`https?://[^\s]+`)
	// Mentions: @ followed by a domain, as Bluesky handles are domain names.
	facetMentionRegex = regexp.MustCompile(`@([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?`)
	// Hashtags: # followed by letters, numbers and underscores.
	facetHashtagRegex = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)
)

// facetRule turns a regexp match into a facet feature; ok is false to skip it.
type facetRule struct {
	re      *regexp.Regexp
	feature func(match string) (feature BlueskyFacetFeature, ok bool)
}

// parseFacets marks up links, then mentions, then hashtags. A later kind never
// overlaps an earlier one, so text inside a link stays part of the link.
func (b *BlueskyConnector) parseFacets(text string) []BlueskyFacet {
	if text == "" {
		return nil
	}
	rules := []facetRule{
		{facetURLRegex, linkFeature},
		{facetMentionRegex, b.mentionFeature},
		{facetHashtagRegex, tagFeature},
	}
	used := make([]bool, len(text))
	facets := []BlueskyFacet{}
	for _, rule := range rules {
		facets = append(facets, findFacets(text, rule, used)...)
	}
	return facets
}

// findFacets returns the rule's facets that don't overlap a used byte range,
// and marks their ranges used. Offsets are bytes, as Bluesky expects.
func findFacets(text string, rule facetRule, used []bool) []BlueskyFacet {
	var facets []BlueskyFacet
	for _, match := range rule.re.FindAllStringIndex(text, -1) {
		start, end := match[0], match[1]
		if slices.Contains(used[start:end], true) {
			continue
		}
		feature, ok := rule.feature(text[start:end])
		if !ok {
			continue
		}
		facets = append(facets, BlueskyFacet{
			Index:    BlueskyFacetIndex{ByteStart: start, ByteEnd: end},
			Features: []BlueskyFacetFeature{feature},
		})
		for i := start; i < end; i++ {
			used[i] = true
		}
	}
	return facets
}

func linkFeature(match string) (BlueskyFacetFeature, bool) {
	parsed, err := url.Parse(match)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return BlueskyFacetFeature{}, false
	}
	return BlueskyFacetFeature{Type: "app.bsky.richtext.facet#link", URI: match}, true
}

// mentionFeature keeps a mention whose handle can't be resolved, without a
// DID; Bluesky may still accept it.
func (b *BlueskyConnector) mentionFeature(match string) (BlueskyFacetFeature, bool) {
	handle := match[1:]
	did, err := b.resolveHandleToDID(handle)
	if err != nil {
		b.Logger().Warnf("Failed to resolve handle %s to DID: %v", handle, err)
	}
	return BlueskyFacetFeature{Type: "app.bsky.richtext.facet#mention", DID: did}, true
}

func tagFeature(match string) (BlueskyFacetFeature, bool) {
	return BlueskyFacetFeature{Type: "app.bsky.richtext.facet#tag", Tag: match[1:]}, true
}

func (b *BlueskyConnector) GetIcon() string {
	return "bi:bluesky"
}

func (b *BlueskyConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	sa, err := b.ensureFreshOAuthToken(socialAccount)
	if err != nil {
		return err
	}
	user, err := b.identityForRefresh(sa)
	if err != nil {
		return err
	}
	if user == "" {
		b.Logger().Warnf("Skipping refresh: no user identity available")
		return nil
	}
	b.maybeResolveHandleOnRefresh(sa, user)
	if strings.HasPrefix(user, "did:") {
		b.Logger().Debugf("User is already a DID, no further resolution needed: %s", user)
		return nil
	}
	b.Logger().Warnf("Unexpected handle in OnRefresh: %s", user)
	return nil
}

func (b *BlueskyConnector) ensureFreshOAuthToken(socialAccount *db.SocialAccount) (*db.SocialAccount, error) {
	if !socialAccount.OAuth2TokenExpiry.Before(time.Now()) {
		return socialAccount, nil
	}
	b.Logger().Warnf("OAuth token is expired, attempting to refresh")
	if err := b.refreshToken(socialAccount); err != nil {
		b.Logger().Errorf("Failed to refresh expired token: %v", err)
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	if updatedAccount, err := b.db.GetSocialAccount(socialAccount.ID); err == nil {
		return updatedAccount, nil
	}
	return socialAccount, nil
}

func (b *BlueskyConnector) identityForRefresh(socialAccount *db.SocialAccount) (string, error) {
	if socialAccount.Identity != "" {
		return socialAccount.Identity, nil
	}
	b.Logger().Warnf("Social account identity is empty, attempting to get DID from token")
	if socialAccount.OAuth2Token == "" {
		b.Logger().Errorf("No OAuth token available to extract DID from")
		return "", fmt.Errorf("no OAuth token available")
	}
	did, err := b.extractDIDFromToken(socialAccount.OAuth2Token)
	if err != nil {
		b.Logger().Errorf("Failed to extract DID from token: %v", err)
		return "", fmt.Errorf("failed to extract DID from token: %w", err)
	}
	b.Logger().Infof("Extracted user DID from token: %s", did)
	// TODO: Implement UpdateSocialAccountDID method in DB
	b.Logger().Warnf("Would update social account DID to: %s (method not implemented)", did)
	return did, nil
}

func (b *BlueskyConnector) maybeResolveHandleOnRefresh(socialAccount *db.SocialAccount, user string) {
	if !strings.HasPrefix(user, "did:") {
		return
	}
	b.Logger().Infof("Have DID but no handle, attempting to resolve handle for DID: %s", user)
	handle, err := b.getHandleFromDID(user)
	if err != nil || handle == "" {
		b.Logger().Warnf("Failed to get handle for DID %s: %v", user, err)
		return
	}
	b.Logger().Infof("Retrieved handle for DID %s: %s", user, handle)
	b.persistResolvedHandle(socialAccount.ID, handle)
}

func (b *BlueskyConnector) persistResolvedHandle(accountID uint32, handle string) {
	if err := b.db.UpdateSocialAccountIdentity(accountID, handle); err != nil {
		b.Logger().Errorf("Failed to update social account identity: %v", err)
	}
}

type blueskyTokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// refreshToken attempts to refresh an expired OAuth token with DPoP and JWT client assertion
func (b *BlueskyConnector) refreshToken(socialAccount *db.SocialAccount) error {
	if socialAccount.OAuth2RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	client, req, err := b.prepareRefreshTokenRequest(socialAccount)
	if err != nil {
		return err
	}
	bodyBytes, err := executeRefreshTokenRequest(client, req, b.Logger())
	if err != nil {
		return err
	}
	tokenResponse, err := parseRefreshTokenSuccess(bodyBytes, b.Logger())
	if err != nil {
		return err
	}
	return b.applyRefreshedToken(socialAccount, tokenResponse)
}

func (b *BlueskyConnector) prepareRefreshTokenRequest(socialAccount *db.SocialAccount) (*utils.ChainingHttpClient, *http.Request, error) {
	baseURL := b.db.GetCvarString(db.CvarKeys.BaseUrl)
	clientId := baseURL + "/oauth/client-metadata.json"
	if clientId == "" {
		return nil, nil, fmt.Errorf("bluesky BASE_URL not configured")
	}
	b.Logger().Infof("Using client_id for token refresh: %s", clientId)
	b.Logger().Infof("Refreshing OAuth token for Bluesky account ID: %d", socialAccount.ID)

	client := b.createDPoPClientForRefresh(socialAccount)
	if client == nil {
		return nil, nil, fmt.Errorf("cannot create DPoP client for token refresh")
	}
	clientAssertion, err := b.createClientAssertion(clientId)
	if err != nil {
		b.Logger().Errorf("Failed to create client assertion: %v", err)
		return nil, nil, fmt.Errorf("failed to create client assertion: %w", err)
	}
	req, err := newRefreshTokenHTTPRequest(socialAccount.OAuth2RefreshToken, clientId, clientAssertion)
	if err != nil {
		b.Logger().Errorf("Failed to create refresh token request: %v", err)
		return nil, nil, fmt.Errorf("token refresh failed: %w", err)
	}
	return client, req, nil
}

func newRefreshTokenHTTPRequest(refreshToken, clientId, clientAssertion string) (*http.Request, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientId)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", clientAssertion)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://bsky.social/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func executeRefreshTokenRequest(client *utils.ChainingHttpClient, req *http.Request, logger interface {
	Errorf(string, ...any)
}) ([]byte, error) {
	resp, err := client.UnderlyingClient().Do(req)
	if err != nil {
		logger.Errorf("Failed to execute refresh token request: %v", err)
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		logger.Errorf("Failed to read response body: %v", readErr)
		return nil, fmt.Errorf("token refresh failed: failed to read response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, refreshTokenHTTPError(resp.StatusCode, bodyBytes, logger)
	}
	return bodyBytes, nil
}

func refreshTokenHTTPError(statusCode int, bodyBytes []byte, logger interface {
	Errorf(string, ...any)
}) error {
	logger.Errorf("Token refresh failed with status %d. Response body: %s", statusCode, string(bodyBytes))
	if err := refreshTokenAPIErrorFromBody(bodyBytes); err != nil {
		return err
	}
	return fmt.Errorf("token refresh failed: unexpected status code: %d", statusCode)
}

func refreshTokenAPIErrorFromBody(bodyBytes []byte) error {
	errorMsg, errorDesc, ok := decodeRefreshTokenErrorFields(bodyBytes)
	if !ok {
		return nil
	}
	if isInvalidRefreshGrant(errorMsg, errorDesc) {
		return fmt.Errorf("refresh token invalid or expired - re-authentication required%s", errorDesc)
	}
	return fmt.Errorf("token refresh failed: %s%s", errorMsg, errorDesc)
}

func decodeRefreshTokenErrorFields(bodyBytes []byte) (msg, desc string, ok bool) {
	var errorResponse map[string]any
	if json.Unmarshal(bodyBytes, &errorResponse) != nil {
		return "", "", false
	}
	errorMsg, ok := errorResponse["error"].(string)
	if !ok {
		return "", "", false
	}
	return errorMsg, refreshTokenErrorDesc(errorResponse), true
}

func refreshTokenErrorDesc(errorResponse map[string]any) string {
	desc, ok := errorResponse["error_description"].(string)
	if !ok {
		return ""
	}
	return ": " + desc
}

func isInvalidRefreshGrant(errorMsg, errorDesc string) bool {
	return errorMsg == "invalid_grant" && strings.Contains(strings.ToLower(errorDesc), "refresh token")
}

func parseRefreshTokenSuccess(bodyBytes []byte, logger interface {
	Errorf(string, ...any)
}) (blueskyTokenRefreshResponse, error) {
	var tokenResponse blueskyTokenRefreshResponse
	if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
		logger.Errorf("Failed to parse token response: %v", err)
		return blueskyTokenRefreshResponse{}, fmt.Errorf("token refresh failed: failed to parse response: %w", err)
	}
	if tokenResponse.AccessToken == "" {
		logger.Errorf("Received empty access token in refresh response")
		return blueskyTokenRefreshResponse{}, fmt.Errorf("received empty access token")
	}
	return tokenResponse, nil
}

func (b *BlueskyConnector) applyRefreshedToken(socialAccount *db.SocialAccount, tokenResponse blueskyTokenRefreshResponse) error {
	expiryTime := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	b.Logger().Infof("Token refreshed successfully, expires at: %v", expiryTime)
	err := b.db.UpdateSocialAccountToken(socialAccount.ID, tokenResponse.AccessToken, tokenResponse.RefreshToken, tokenResponse.ExpiresIn)
	if err != nil {
		b.Logger().Errorf("Failed to update social account token: %v", err)
		return fmt.Errorf("failed to update token in database: %w", err)
	}
	socialAccount.OAuth2Token = tokenResponse.AccessToken
	socialAccount.OAuth2RefreshToken = tokenResponse.RefreshToken
	socialAccount.OAuth2TokenExpiry = expiryTime
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
// appViewURL is the public AppView used to resolve handles. Tests replace it.
var appViewURL = "https://api.bsky.app"

func (b *BlueskyConnector) resolveHandleToDID(handle string) (string, error) {
	// Use public AppView for handle resolution (no auth required)
	url := fmt.Sprintf("%s/xrpc/com.atproto.identity.resolveHandle?handle=%s", appViewURL, handle)

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
	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid DID format: %s", did)
	}
	method, identifier := parts[1], parts[2]
	b.Logger().Debugf("Resolving DID document for method: %s, identifier: %s", method, identifier)
	return b.resolveDIDDocumentByMethod(method, identifier, parts)
}

func (b *BlueskyConnector) resolveDIDDocumentByMethod(method, identifier string, parts []string) (map[string]any, error) {
	switch method {
	case "plc":
		return b.resolvePLCDIDDocument(identifier)
	case "web":
		return b.resolveWebDIDDocument(identifier)
	case "did":
		return b.resolveNestedDIDDocument(parts)
	default:
		return nil, fmt.Errorf("unsupported DID method: %s", method)
	}
}

func (b *BlueskyConnector) resolveNestedDIDDocument(parts []string) (map[string]any, error) {
	if len(parts) >= 4 {
		return b.resolveDIDDocument(strings.Join(parts[1:], ":"))
	}
	return nil, fmt.Errorf("unsupported DID method: did")
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
	services, ok := didDoc["service"].([]any)
	if !ok {
		return "", fmt.Errorf("no services found in DID document")
	}
	if endpoint := findExactPDSService(services); endpoint != "" {
		b.Logger().Debugf("Found AtprotoPersonalDataServer service: %s", endpoint)
		return endpoint, nil
	}
	if endpoint, serviceType := findRelatedPDSService(services); endpoint != "" {
		b.Logger().Debugf("Found atproto-related service: %s -> %s", serviceType, endpoint)
		return endpoint, nil
	}
	return "", fmt.Errorf("no AtprotoPersonalDataServer service found in DID document")
}

func serviceMapFields(service any) (serviceType, endpoint string, ok bool) {
	serviceMap, ok := service.(map[string]any)
	if !ok {
		return "", "", false
	}
	serviceType, ok = serviceMap["type"].(string)
	if !ok {
		return "", "", false
	}
	endpoint, ok = serviceMap["serviceEndpoint"].(string)
	if !ok {
		return "", "", false
	}
	return serviceType, endpoint, true
}

func findExactPDSService(services []any) string {
	for _, service := range services {
		serviceType, endpoint, ok := serviceMapFields(service)
		if ok && serviceType == "AtprotoPersonalDataServer" {
			return endpoint
		}
	}
	return ""
}

func isRelatedPDSServiceType(serviceType string) bool {
	lower := strings.ToLower(serviceType)
	return strings.Contains(lower, "atproto") || strings.Contains(lower, "pds") || strings.Contains(lower, "bsky")
}

func findRelatedPDSService(services []any) (endpoint, serviceType string) {
	for _, service := range services {
		st, ep, ok := serviceMapFields(service)
		if ok && isRelatedPDSServiceType(st) {
			return ep, st
		}
	}
	return "", ""
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
	b.checkOAuth2RedirectURL(res, baseURL)
	if !b.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible) {
		res.AddSettingsIssue("Bluesky OAuth requires Is Publicly Accessible to be enabled.", db.CvarKeys.IsPubliclyAccessible)
	}
	return res
}

func (b *BlueskyConnector) checkOAuth2RedirectURL(res *connector.ConfigurationCheckResult, baseURL string) {
	redirectURL := b.db.GetCvarString(db.CvarKeys.OAuth2RedirectURL)
	if redirectURL == "" {
		res.AddSettingsIssue("OAuth2 Redirect URL is not set.", db.CvarKeys.OAuth2RedirectURL)
		return
	}
	if baseURL != "" && redirectURL != baseURL+"/oauth2callback" {
		res.AddSettingsIssue("OAuth2 Redirect URL should match your base URL plus /oauth2callback.", db.CvarKeys.OAuth2RedirectURL)
	}
}

type dpopTransport struct {
	base              http.RoundTripper
	dpopNonce         string
	signedAccessToken string
	accessTokenHash   string
	privKey           *ecdsa.PrivateKey
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.sign(req)
	resp, err := t.base.RoundTrip(req)
	if isDPoPErrorStatus(resp) {
		if handled, done, handledErr := t.handleErrorResponse(req, resp); done {
			return handled, handledErr
		}
	}
	t.rememberNonce(resp)
	return resp, err
}

// sign adds a DPoP proof for this request, and the access token if there is one.
func (t *dpopTransport) sign(req *http.Request) {
	if t.accessTokenHash == "" && t.signedAccessToken != "" {
		t.accessTokenHash = calculateAccessTokenHash(t.signedAccessToken)
	}
	proof, _ := createDPoPProof(t.privKey, req.Method, req.URL.String(), t.dpopNonce, t.accessTokenHash)
	req.Header.Set("DPoP", proof)
	if t.signedAccessToken != "" {
		req.Header.Set("Authorization", "DPoP "+t.signedAccessToken)
	}
}

func isDPoPErrorStatus(resp *http.Response) bool {
	return resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest)
}

// handleErrorResponse retries once with the server's nonce when it asks for
// one, or reports an invalid token. Otherwise done is false and resp, with its
// body still readable, is the result.
func (t *dpopTransport) handleErrorResponse(req *http.Request, resp *http.Response) (*http.Response, bool, error) {
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	if retried := t.retryWithNonce(req, resp, body); retried != nil {
		return retried, true, nil
	}
	if isTokenError(body) {
		return resp, true, fmt.Errorf("invalid or expired OAuth token")
	}
	return nil, false, nil
}

// retryWithNonce returns the retried response, or nil if the server didn't ask
// for a nonce or the retry failed.
func (t *dpopTransport) retryWithNonce(req *http.Request, resp *http.Response, body []byte) *http.Response {
	if !strings.Contains(string(body), "use_dpop_nonce") {
		return nil
	}
	nonce := resp.Header.Get("DPoP-Nonce")
	if nonce == "" {
		return nil
	}
	t.dpopNonce = nonce
	retried, err := t.base.RoundTrip(t.cloneForRetry(req))
	if err != nil || retried == nil {
		return nil
	}
	return retried
}

func (t *dpopTransport) cloneForRetry(req *http.Request) *http.Request {
	retry := req.Clone(req.Context())
	if req.Body != nil && req.GetBody != nil {
		if body, err := req.GetBody(); err == nil {
			retry.Body = body
		}
	}
	t.sign(retry)
	return retry
}

// isTokenError recognises an expired token, or a token used somewhere other than the PDS.
func isTokenError(body []byte) bool {
	return bytes.Contains(body, []byte("InvalidToken")) || bytes.Contains(body, []byte("OAuth tokens are meant for PDS access only"))
}

// rememberNonce keeps the server's latest nonce for the next request.
func (t *dpopTransport) rememberNonce(resp *http.Response) {
	if resp == nil {
		return
	}
	if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
		t.dpopNonce = nonce
	}
}

func (b *BlueskyConnector) OnOAuth2Callback(code string, verifier string, headers map[string]string) error {
	privKey, err := generatePrivateKey()
	if err != nil {
		b.Logger().Errorf("Error creating DPoP key: %v", err)
		return err
	}
	tp := b.bindTempDPoPClient(privKey)
	token, err := b.exchangeOAuthCode(code, verifier)
	if err != nil {
		return err
	}
	tp.signedAccessToken = token.AccessToken
	b.Logger().Debugf("Received token on exchange (token type: %s, expires in: %v)", token.TokenType, time.Until(token.Expiry))

	identity := b.identityFromAccessToken(token.AccessToken)
	handle := b.handleFromIdentity(identity)
	storedKey, err := encodeStoredDPoPKey(privKey)
	if err != nil {
		return err
	}
	return b.db.RegisterAccount(&db.SocialAccount{
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
}

func (b *BlueskyConnector) bindTempDPoPClient(privKey *ecdsa.PrivateKey) *dpopTransport {
	client := utils.NewClient(b.Logger())
	tp := &dpopTransport{
		base:    client.UnderlyingClient().Transport,
		privKey: privKey,
	}
	client.UnderlyingClient().Transport = tp
	b.tmpClient = client
	return tp
}

func (b *BlueskyConnector) exchangeOAuthCode(code, verifier string) (*oauth2.Token, error) {
	config := b.GetOAuth2Config()
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, b.tmpClient.UnderlyingClient())
	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		b.Logger().Infof("Error exchanging code: %v", err)
		return nil, err
	}
	return token, nil
}

func (b *BlueskyConnector) identityFromAccessToken(accessToken string) string {
	if accessToken == "" {
		return ""
	}
	did, didErr := b.extractDIDFromToken(accessToken)
	if didErr != nil {
		b.Logger().Warnf("Failed to extract DID from token: %v", didErr)
		return ""
	}
	b.Logger().Infof("Extracted user DID from token: %s", did)
	return did
}

func (b *BlueskyConnector) handleFromIdentity(identity string) string {
	if identity == "" {
		return ""
	}
	h, handleErr := b.getHandleFromDID(identity)
	if handleErr != nil || h == "" {
		b.Logger().Warnf("Failed to get handle for DID %s: %v", identity, handleErr)
		return ""
	}
	b.Logger().Infof("Retrieved handle for DID %s: %s", identity, h)
	return h
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
	return newDPoPClient(b.Logger(), priv, sa.OAuth2Token)
}

func newDPoPClient(logger *log.Entry, priv *ecdsa.PrivateKey, accessToken string) *utils.ChainingHttpClient {
	client := utils.NewClient(logger)
	tp := &dpopTransport{
		base:              client.UnderlyingClient().Transport,
		privKey:           priv,
		signedAccessToken: accessToken,
		accessTokenHash:   calculateAccessTokenHash(accessToken),
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
	plcURL := fmt.Sprintf("https://plc.directory/%s", did)
	client := utils.NewClient(b.Logger())
	var didDoc map[string]any
	client.Get(plcURL).AsJson(&didDoc)
	if client.Err != nil {
		return "", fmt.Errorf("failed to get PLC document: %w", client.Err)
	}
	return handleFromAlsoKnownAs(didDoc)
}

func handleFromAlsoKnownAs(didDoc map[string]any) (string, error) {
	alsoKnownAs, ok := didDoc["alsoKnownAs"].([]any)
	if !ok || len(alsoKnownAs) == 0 {
		return "", fmt.Errorf("no alsoKnownAs found in PLC document")
	}
	firstEntry, ok := alsoKnownAs[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid alsoKnownAs format")
	}
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
