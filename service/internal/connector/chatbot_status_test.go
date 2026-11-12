package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatBotStatusMessage(t *testing.T) {
	assert.Equal(t, "Bot is running and connected", ChatBotStatusMessage(true, "boom", "starting"))
	assert.Equal(t, "boom", ChatBotStatusMessage(false, "boom", "starting"))
	assert.Equal(t, "starting", ChatBotStatusMessage(false, "", "starting"))
	assert.Equal(t, "Stopped (not connected)", ChatBotStatusMessage(false, "", ""))
}
