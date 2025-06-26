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

func ClientDoJson(client *http.Client, req *http.Request, v any) error {
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("error decoding JSON response: %w", err)
		}
	}

	return nil
}

func NewHttpClientAndGetReqWithUrlEncodedMap(requrl string, token string, body map[string]string) (*http.Client, *http.Request, error) {
	form := url.Values{}
	for key, value := range body {
		form.Add(key, value)
	}

	req, err := http.NewRequest("POST", requrl, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Basic "+token)
		//		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{
		Transport: NewLoggingTransport(nil),
	}

	return client, req, nil
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
