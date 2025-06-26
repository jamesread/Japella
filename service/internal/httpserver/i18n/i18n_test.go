package i18n

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLanguageDir(t *testing.T) {
	dir := getLanguageDir()
	assert.NotEmpty(t, dir, "Expected a non-empty language directory")
}
