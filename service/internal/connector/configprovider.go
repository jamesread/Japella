package connector

import (
	"github.com/jamesread/japella/internal/db"
)

type ConfigProvider interface {
	GetCvars() map[string]*db.Cvar
	OnCvarChange(key string, value string)
	CheckConfiguration() *ConfigurationCheckResult
}

type ConfigurationCheckResult struct {
	Issues []string
}

func (c *ConfigurationCheckResult) AddIssue(issue string) {
	c.Issues = append(c.Issues, issue)
}
