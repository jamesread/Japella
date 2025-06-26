package i18n

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGetLanguageDir(t *testing.T) {
	dir := getLanguageDir()
	assert.NotEmpty(t, dir, "Expected a non-empty language directory")
}
