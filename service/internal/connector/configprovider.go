package connector

import (
	"github.com/jamesread/japella/internal/db"
)

type ConfigProvider interface {
	GetCvars() map[string]*db.Cvar
	OnCvarChange(key string, value string)
	CheckConfiguration() *ConfigurationCheckResult
}

type ConfigurationIssue struct {
	Message   string
	FixPath   string
	FixHash   string
	FixLabel  string
	FixAction string
}

type ConfigurationCheckResult struct {
	Issues []ConfigurationIssue
}

const FixActionRegisterClient = "register_client"

func (c *ConfigurationCheckResult) AddIssue(issue string) {
	c.Issues = append(c.Issues, ConfigurationIssue{Message: issue})
}

func (c *ConfigurationCheckResult) AddSettingsIssue(message string, cvarKey string) {
	c.Issues = append(c.Issues, ConfigurationIssue{
		Message:  message,
		FixPath:  "/settings",
		FixHash:  cvarKey,
		FixLabel: "Open Settings",
	})
}

func (c *ConfigurationCheckResult) AddRouteIssue(message, path, label string) {
	c.Issues = append(c.Issues, ConfigurationIssue{
		Message:  message,
		FixPath:  path,
		FixLabel: label,
	})
}

func (c *ConfigurationCheckResult) AddRegisterClientIssue(message string) {
	c.Issues = append(c.Issues, ConfigurationIssue{
		Message:   message,
		FixAction: FixActionRegisterClient,
		FixLabel:  "Register OAuth application",
	})
}
