package connector

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationCheckResult(t *testing.T) {
	res := ConfigurationCheckResult{}
	res.AddIssue("This is an issue")
	res.AddSettingsIssue("Set the client ID in settings.", "x.client_id")
	res.AddRegisterClientIssue("Register the OAuth application.")

	assert.Equal(t, 3, len(res.Issues), "Expected three issues in the result")
	assert.Equal(t, "/settings", res.Issues[1].FixPath)
	assert.Equal(t, "x.client_id", res.Issues[1].FixHash)
	assert.Equal(t, FixActionRegisterClient, res.Issues[2].FixAction)
}
