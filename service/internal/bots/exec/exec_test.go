package exec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCommandContextConfinesScriptToScriptDir(t *testing.T) {
	cmd, err := getCommandContext(t.Context(), "../../bin/sh", "")
	require.NoError(t, err)
	assert.Equal(t, "/usr/libexec/japella/sh", cmd.Path)
}

func TestGetCommandContextSplitsArguments(t *testing.T) {
	cmd, err := getCommandContext(t.Context(), "uptime", `-v "two words" 1.2.3!`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/usr/libexec/japella/uptime", "-v", "two words", "123"}, cmd.Args)
}
