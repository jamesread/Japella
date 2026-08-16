package commands

import (
	"fmt"

	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
)

func openDB() (*db.DB, error) {
	cfg := runtimeconfig.Get()
	database := &db.DB{}
	database.SetDatabaseConfig(cfg.Database)
	database.ReconnectDatabaseAndSetErrorMessage()
	if !database.ConnEstablished() {
		msg := database.GetErrorMessage()
		if msg == "" {
			msg = "connection not established"
		}
		return nil, fmt.Errorf("database: %s", msg)
	}
	return database, nil
}
