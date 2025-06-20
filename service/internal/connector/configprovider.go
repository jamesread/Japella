package connector

import (
	"github.com/jamesread/japella/internal/db"
)

type ConfigProvider interface {
	GetCvars() (map[string]*db.Cvar)
	OnCvarChange(key string, value string)
}
