package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	logp "github.com/sirupsen/logrus"

	"net/url"

	"github.com/jamesread/japella/internal/debuglog"
)

type ChainingHttpClient struct {
	rt http.RoundTripper

	client *http.Client

	url string
	Err error
	req *http.Request
	Res *http.Response
	// ResBody is the response body when Err != nil (e.g. non-2xx), for logging/debugging
	ResBody []byte

	logger *logp.Entry
}

func (l *ChainingHttpClient) UnderlyingClient() *http.Client {
	return l.client
}

func (l *ChainingHttpClient) logRequestHeaders(req *http.Request) {
	if !debuglog.HTTPEnabled() {
		return
	}
	for key, values := range req.Header {
		for _, value := range values {
			debuglog.HTTPf("Request Header: %s: %s", key, value)
		}
	}
}

func (l *ChainingHttpClient) logResponseHeaders(resp *http.Response) {
	if !debuglog.HTTPEnabled() {
		return
	}
	for key, values := range resp.Header {
		for _, value := range values {
			debuglog.HTTPf("Response Header: %s: %s", key, value)
		}
	}
}

func (l *ChainingHttpClient) RoundTrip(req *http.Request) (*http.Response, error) {
	debuglog.HTTPf("Request: %s %s", req.Method, req.URL.String())
	l.logRequestHeaders(req)
	if err := l.replayAndLogRequestBody(req); err != nil {
		return nil, err
	}

	resp, err := l.rt.RoundTrip(req)
	if err != nil {
		l.logger.Errorf("Error during request: %v", err)
		return nil, err
	}

	debuglog.HTTP(logp.Fields{
		"status": resp.Status,
		"url":    req.URL.String(),
		"method": req.Method,
	}, "HTTP Response")
	l.logResponseHeaders(resp)

	// use strings.Contains here because some servers include the charset in the Content-Type, so it does not exactly match "application/json"
	isJson := strings.Contains(resp.Header.Get("Content-Type"), "application/json")
	bodyString, err := ReadBody(resp)
	if err != nil {
		l.logger.Errorf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	l.logBodyContent(isJson, bodyString)
	return resp, nil
}

func (l *ChainingHttpClient) replayAndLogRequestBody(req *http.Request) error {
	if req.Body == nil {
		return nil
	}
	contentType := req.Header.Get("Content-Type")
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	bodyBytes := buf.Bytes()
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	logRequestBody(contentType, n, bodyBytes)
	return nil
}

func logRequestBody(contentType string, n int64, bodyBytes []byte) {
	if !debuglog.HTTPEnabled() {
		return
	}
	if shouldOmitRequestBodyLog(contentType, n) {
		debuglog.HTTPf("Request body: %d bytes (Content-Type: %s), omitted from log", n, contentType)
		return
	}
	debuglog.HTTPf("Request Body Content:\n%s", string(bodyBytes))
}

func shouldOmitRequestBodyLog(contentType string, n int64) bool {
	return n > 1024 ||
		strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "application/octet-stream") ||
		strings.HasPrefix(contentType, "multipart/")
}

func ReadBody(r *http.Response) (string, error) {
	bodyString := ""

	if r.Body != nil {
		buf := new(bytes.Buffer)

		_, err := buf.ReadFrom(r.Body)

		if err != nil {
			return "", fmt.Errorf("error reading response body: %w", err)
		}

		bodyString = buf.String()
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBufferString(bodyString)) // Reassign the body to allow further reading
	}

	return bodyString, nil
}

func (l *ChainingHttpClient) logBodyContent(isJson bool, bodyString string) {
	if !debuglog.HTTPEnabled() {
		return
	}
	if isJson {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, []byte(bodyString), "", "  "); err != nil {
			debuglog.HTTPf("Response Body Content: %s", bodyString)
			return
		}
		debuglog.HTTPf("Response Body JSON:\n%s", prettyJSON.String())
		return
	}
	debuglog.HTTPf("Response Body Content: %s", bodyString)
}

// ReadBodyNoChange returns the current response body as string without modifying l.Res
func ReadBodyNoChange(r *http.Response) string {
	if r == nil || r.Body == nil {
		return ""
	}
	// Duplicate the body reader by reading it, but we cannot without consuming; rely on callers that just need a one-off
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r.Body); err != nil {
		logp.Warnf("Response body read was incomplete: %v", err)
	}
	body := buf.String()
	_ = r.Body.Close()
	r.Body = io.NopCloser(strings.NewReader(body))
	return body
}

func NewLoggingTransport(rt http.RoundTripper, logger *logp.Entry) *ChainingHttpClient {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &ChainingHttpClient{
		rt:     rt,
		logger: logger,
	}
}

// Social network APIs occasionally accept a connection and never answer; without
// a timeout that would block the calling job forever.
const chainingClientTimeout = 2 * time.Minute

func NewClient(logger *logp.Entry) *ChainingHttpClient {
	logger = logger.WithField("client", uuid.New().String())

	client := &ChainingHttpClient{
		rt:     http.DefaultTransport,
		logger: logger,
		client: &http.Client{
			Transport: NewLoggingTransport(nil, logger),
			Timeout:   chainingClientTimeout,
		},
	}

	return client
}

func (c *ChainingHttpClient) AsJson(v any) {
	c.Res, c.Err = c.client.Do(c.req) //nolint:bodyclose // closed by the defer below once Err is checked
	if c.Err != nil {
		c.Err = fmt.Errorf("error making request: %w", c.Err)
		return
	}
	defer func() { _ = c.Res.Body.Close() }()
	if !isHTTPSuccess(c.Res.StatusCode) {
		c.ResBody, _ = io.ReadAll(c.Res.Body)
		c.Err = fmt.Errorf("unexpected status code: %d", c.Res.StatusCode)
		return
	}
	c.decodeJSONBody(v)
}

func isHTTPSuccess(status int) bool {
	return status >= 200 && status < 300
}

func (c *ChainingHttpClient) decodeJSONBody(v any) {
	if v == nil {
		return
	}
	if err := json.NewDecoder(c.Res.Body).Decode(v); err != nil {
		c.Err = fmt.Errorf("error decoding JSON response: %w", err)
	}
}

func (c *ChainingHttpClient) PostWithFormVars(requrl string, body map[string]string) *ChainingHttpClient {
	c.url = requrl

	form := url.Values{}
	for key, value := range body {
		form.Add(key, value)
	}

	c.req, c.Err = http.NewRequestWithContext(context.Background(), http.MethodPost, c.url, strings.NewReader(form.Encode()))
	if c.Err != nil {
		return c
	}
	c.req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c
}

func (c *ChainingHttpClient) Get(requrl string) *ChainingHttpClient {
	c.url = requrl

	var err error
	c.req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, c.url, nil)

	if err != nil {
		c.Err = fmt.Errorf("error creating request: %w", err)
	}

	return c
}

func (c *ChainingHttpClient) WithBasicAuth(token string) *ChainingHttpClient {
	c.req.Header.Set("Authorization", "Basic "+token)
	return c
}

func (c *ChainingHttpClient) PostWithJson(requrl string, body any) *ChainingHttpClient {
	jsonBody, err := json.MarshalIndent(body, "", "  ")

	if err != nil {
		c.Err = fmt.Errorf("error encoding body to JSON: %w", err)
		return c
	}

	c.req, c.Err = http.NewRequestWithContext(context.Background(), http.MethodPost, requrl, bytes.NewReader(jsonBody))
	if c.Err != nil {
		return c
	}
	// Allow body to be recreated on retry (e.g., DPoP nonce retry)
	reqBody := append([]byte(nil), jsonBody...)
	c.req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(reqBody)), nil
	}
	c.req.Header.Set("Content-Type", "application/json")

	if c.Err != nil {
		c.Err = fmt.Errorf("error creating request: %w", c.Err)
		return c
	}

	return c
}

func (c *ChainingHttpClient) WithBearerToken(token string) *ChainingHttpClient {
	if token != "" {
		c.req.Header.Set("Authorization", "Bearer "+token)
	}

	return c
}
