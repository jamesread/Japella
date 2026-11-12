package frontend

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestWebuiFileExists(t *testing.T) {
	webui := fstest.MapFS{
		"index.html":        {Data: []byte("app")},
		"assets/app.js":     {Data: []byte("js")},
		"docs/index.html":   {Data: []byte("docs")},
		"empty/placeholder": {Data: []byte("")},
	}

	assert.True(t, webuiFileExists(webui, "/"))
	assert.True(t, webuiFileExists(webui, "/assets/app.js"))
	assert.True(t, webuiFileExists(webui, "/docs"))
	assert.False(t, webuiFileExists(webui, "/empty"))
	assert.False(t, webuiFileExists(webui, "/timeline"))
	assert.False(t, webuiFileExists(webui, "/../../etc/passwd"))
}
