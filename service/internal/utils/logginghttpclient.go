package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"net/url"
)

type loggingHTTPClient struct {
	rt http.RoundTripper

	client *http.Client

	url string
	Err error
	req *http.Request
}

func (l *loggingHTTPClient) logRequestHeaders(req *http.Request) {
	for key, values := range req.Header {
		for _, value := range values {
			log.Infof("Request Header: %s: %s", key, value)
		}
	}
}

func (l *loggingHTTPClient) logResponseHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			log.Infof("Response Header: %s: %s", key, value)
		}
	}
}

func (l *loggingHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Infof("Request: %s %s", req.Method, req.URL.String())

	l.logRequestHeaders(req)

	if req.Body != nil {
		log.Debugf("Request Body: %v", req.Body)

		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		bodyString := buf.String()
		req.Body.Close()                                           // Close the body to avoid resource leaks
		req.Body = io.NopCloser(bytes.NewBufferString(bodyString)) // Reassign the body to allow further reading

		fmt.Printf("Request Body Content:\n%s\n", bodyString)
	}

	resp, err := l.rt.RoundTrip(req)

	if err != nil {
		log.Errorf("Error during request: %v", err)
		return nil, err
	}

	log.Infof("Response: %d %s", resp.StatusCode, resp.Status)

	l.logResponseHeaders(resp)

	// use strings.Contains here because some servers include the charset in the Content-Type, so it does not exactly match "application/json"
	isJson := strings.Contains(resp.Header.Get("Content-Type"), "application/json")

	bodyString := ""
	if resp.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		bodyString = buf.String()
		resp.Body.Close()                                           // Close the body to avoid resource leaks
		resp.Body = io.NopCloser(bytes.NewBufferString(bodyString)) // Reassign the body to allow further reading
	}

	l.logBodyContent(isJson, bodyString)

	return resp, nil
}

func (l *loggingHTTPClient) logBodyContent(isJson bool, bodyString string) {
	if log.IsLevelEnabled(log.DebugLevel) {
		if isJson {
			var prettyJSON bytes.Buffer

			if err := json.Indent(&prettyJSON, []byte(bodyString), "", "  "); err != nil {
				log.Errorf("Error pretty printing JSON response: %v", err)
				log.Debugf("Response Body Content: %s", bodyString)
			}

			fmt.Printf("Response Body JSON:\n%s\n", prettyJSON.String())
		} else {
			log.Debugf("Response Body Content: %s", bodyString)
		}
	}
}

func NewLoggingTransport(rt http.RoundTripper) *loggingHTTPClient {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &loggingHTTPClient{rt: rt}
}

func (c *loggingHTTPClient) AsJson(v any) {
	var resp *http.Response

	resp, c.Err = c.client.Do(c.req)

	if c.Err != nil {
		c.Err = fmt.Errorf("error making request: %w", c.Err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			c.Err = fmt.Errorf("error decoding JSON response: %w", err)
		}
	}
}

func NewClient() *loggingHTTPClient {
	return &loggingHTTPClient{
		rt: http.DefaultTransport,
		client: &http.Client{
			Transport: NewLoggingTransport(nil),
		},
	}
}

func (c *loggingHTTPClient) GetWithFormVars(requrl string, body map[string]string) (*loggingHTTPClient) {
	c.url = requrl

	form := url.Values{}
	for key, value := range body {
		form.Add(key, value)
	}

	c.req, c.Err = http.NewRequest("GET", c.url, strings.NewReader(form.Encode()))
	c.req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c;
}

func (c *loggingHTTPClient) Get(requrl string) *loggingHTTPClient {
	c.url = requrl

	var err error
	c.req, err = http.NewRequest("GET", c.url, nil)

	if err != nil {
		c.Err = fmt.Errorf("error creating request: %w", err)
	}

	return c
}

func (c *loggingHTTPClient) WithBasicAuth(token string) *loggingHTTPClient {
	c.req.Header.Set("Authorization", "Basic "+token)
	return c
}

func NewHttpClientAndGetReqWithUrlEncodedMap(requrl string, token string, body map[string]string) (*http.Client, *http.Request, error) {
	x := NewClient().GetWithFormVars(requrl, body).WithBasicAuth(token)

	return x.client, x.req, x.Err
}

func (c *loggingHTTPClient) PostWithJson(requrl string, body any) (*loggingHTTPClient) {
	jsonBody, err := json.MarshalIndent(body, "", "  ")

	if err != nil {
		c.Err = fmt.Errorf("error encoding body to JSON: %w", err)
		return c
	}

	c.req, c.Err = http.NewRequest("POST", requrl, bytes.NewReader(jsonBody))
	c.req.Header.Set("Content-Type", "application/json")

	if c.Err != nil {
		c.Err = fmt.Errorf("error creating request: %w", c.Err)
		return c
	}

	return c
}

func NewHttpClientAndGetReqWithJson(url string, token string, body any) (*http.Client, *http.Request, error) {
	jsonBody, err := json.MarshalIndent(body, "", "  ")

	if err != nil {
		return nil, nil, fmt.Errorf("error encoding body to JSON: %w", err)
	}

	fmt.Println("JSON Body:\n", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{
		Transport: NewLoggingTransport(nil),
	}

	return client, req, nil
}

func (c *loggingHTTPClient) WithBearerToken(token string) (*loggingHTTPClient) {
	if token != "" {
		c.req.Header.Set("Authorization", "Bearer "+token)
	}

	return c
}

func NewHttpClientAndGetReq(url string, token string) (*http.Client, *http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{
		Transport: NewLoggingTransport(nil),
	}

	return client, req, nil
}
