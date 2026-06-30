package connector

import (
	"fmt"

	"github.com/jamesread/japella/internal/db"
)

// LogChatBotStartupFailure persists a chat bot startup failure to table_logs for the UI logs view.
func LogChatBotStartupFailure(database *db.DB, protocol, identity, errMsg string) {
	if database == nil || errMsg == "" {
		return
	}

	msg := fmt.Sprintf("Chat bot startup failed for %s", protocol)
	if identity != "" {
		msg += fmt.Sprintf(" (%s)", identity)
	}
	msg += ": " + errMsg

	_ = database.InsertTableLog(msg, "error", nil)
}
