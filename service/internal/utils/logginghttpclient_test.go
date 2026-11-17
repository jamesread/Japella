package utils

import (
	"testing"

	logp "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestChainingHttpClientInvalidURLReturnsError(t *testing.T) {
	logger := logp.NewEntry(logp.New())

	cases := map[string]func(*ChainingHttpClient) *ChainingHttpClient{
		"Get": func(c *ChainingHttpClient) *ChainingHttpClient { return c.Get("://bad") },
		"PostWithFormVars": func(c *ChainingHttpClient) *ChainingHttpClient {
			return c.PostWithFormVars("://bad", map[string]string{"a": "b"})
		},
		"PostWithJson": func(c *ChainingHttpClient) *ChainingHttpClient {
			return c.PostWithJson("://bad", map[string]string{"a": "b"})
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			var c *ChainingHttpClient
			assert.NotPanics(t, func() { c = build(NewClient(logger)) })
			assert.Error(t, c.Err)
		})
	}
}

func TestChainingHttpClientHasTimeout(t *testing.T) {
	c := NewClient(logp.NewEntry(logp.New()))
	assert.Positive(t, c.UnderlyingClient().Timeout)
}
