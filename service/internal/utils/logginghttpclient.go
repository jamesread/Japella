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

type ChainingHttpClient struct {
	rt http.RoundTripper

	client *http.Client

	url string
	Err error
	req *http.Request
	Res *http.Response
}

func (l *ChainingHttpClient) UnderlyingClient() *http.Client {
	return l.client
}

func (l *ChainingHttpClient) logRequestHeaders(req *http.Request) {
	for key, values := range req.Header {
		for _, value := range values {
			log.Debugf("Request Header: %s: %s", key, value)
		}
	}
}

func (l *ChainingHttpClient) logResponseHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			log.Debugf("Response Header: %s: %s", key, value)
		}
	}
}

func (l *ChainingHttpClient) RoundTrip(req *http.Request) (*http.Response, error) {
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

	log.WithFields(log.Fields{
		"status":     resp.Status,
		"url":        req.URL.String(),
		"method":     req.Method,
	}).Infof("HTTP Response")

	l.logResponseHeaders(resp)

	// use strings.Contains here because some servers include the charset in the Content-Type, so it does not exactly match "application/json"
	isJson := strings.Contains(resp.Header.Get("Content-Type"), "application/json")

	bodyString, err := ReadBody(resp)

	if err != nil {
		log.Errorf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	l.logBodyContent(isJson, bodyString)

	return resp, nil
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
		r.Body.Close()                                           // Close the body to avoid resource leaks
		r.Body = io.NopCloser(bytes.NewBufferString(bodyString)) // Reassign the body to allow further reading
	}

	return bodyString, nil
}

func (l *ChainingHttpClient) logBodyContent(isJson bool, bodyString string) {
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

func NewLoggingTransport(rt http.RoundTripper) *ChainingHttpClient {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &ChainingHttpClient{rt: rt}
}

func NewClient() *ChainingHttpClient {
	return &ChainingHttpClient{
		rt: http.DefaultTransport,
		client: &http.Client{
			Transport: NewLoggingTransport(nil),
		},
	}
}


func (c *ChainingHttpClient) AsJson(v any) {
	c.Res, c.Err = c.client.Do(c.req)

	if c.Err != nil {
		c.Err = fmt.Errorf("error making request: %w", c.Err)
		return
	}

	defer c.Res.Body.Close()

	if c.Res.StatusCode < 200 || c.Res.StatusCode >= 300 {
		c.Err = fmt.Errorf("unexpected status code: %d", c.Res.StatusCode)
		return
	}

	if v != nil {
		if err := json.NewDecoder(c.Res.Body).Decode(v); err != nil {
			c.Err = fmt.Errorf("error decoding JSON response: %w", err)
		}
	}
}

func (c *ChainingHttpClient) PostWithFormVars(requrl string, body map[string]string) (*ChainingHttpClient) {
	c.url = requrl

	form := url.Values{}
	for key, value := range body {
		form.Add(key, value)
	}

	c.req, c.Err = http.NewRequest("POST", c.url, strings.NewReader(form.Encode()))
	c.req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c;
}

func (c *ChainingHttpClient) Get(requrl string) *ChainingHttpClient {
	c.url = requrl

	var err error
	c.req, err = http.NewRequest("GET", c.url, nil)

	if err != nil {
		c.Err = fmt.Errorf("error creating request: %w", err)
	}

	return c
}

func (c *ChainingHttpClient) WithBasicAuth(token string) *ChainingHttpClient {
	c.req.Header.Set("Authorization", "Basic "+token)
	return c
}

func (c *ChainingHttpClient) PostWithJson(requrl string, body any) (*ChainingHttpClient) {
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

func (c *ChainingHttpClient) WithBearerToken(token string) (*ChainingHttpClient) {
	if token != "" {
		c.req.Header.Set("Authorization", "Bearer "+token)
	}

	return c
}
