// Package health holds Japella's domain-specific health checks; the engine that runs them,
// stores issues and sends notifications is github.com/jamesread/armature-health.
package health

import (
	"context"

	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/armature-health/checks"
	"github.com/jamesread/armature-health/errorlog"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

// ConnectorSource returns the currently started connectors by ID.
type ConnectorSource interface {
	GetServices() map[string]connector.BaseConnector
}

// Deps are the things Japella's health checks inspect.
type Deps struct {
	DB                       *db.DB
	Connectors               ConnectorSource
	ErrorLog                 *errorlog.ErrorLog
	AdminUsesDefaultPassword func(context.Context) (bool, error)
	Getenv                   func(string) string
	LogLevel                 string
}

var riskyEnvVars = []checks.RiskyEnvVar{
	{
		Name:   "JAPELLA_RESET_ADMIN_PASSWORD",
		Detail: `The admin password is reset to "admin" every time Japella starts. Unset the environment variable once you have regained access.`,
	},
}

// DefaultChecks returns the checks Japella runs on startup and periodically.
func DefaultChecks(deps Deps) []armhealth.Check {
	return []armhealth.Check{
		checks.DefaultPassword{
			UsesDefaultPassword: deps.AdminUsesDefaultPassword,
			Product:             "Japella",
			Username:            "admin",
			Link:                "/change-password",
		},
		checks.RiskySettings{
			Getenv:          deps.Getenv,
			LogLevel:        deps.LogLevel,
			LogLevelSetting: "logLevel in config.yaml",
			EnvVars:         riskyEnvVars,
		},
		ConnectorConfigCheck{Connectors: deps.Connectors},
		ChatBotsFailingCheck{Connectors: deps.Connectors},
		KeywordWatchesFailingCheck{DB: deps.DB},
		SocialAccountsExpiredTokensCheck{DB: deps.DB},
		errorlog.RecentErrorsCheck{Log: deps.ErrorLog},
	}
}
