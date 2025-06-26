package connector

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationCheckResult(t *testing.T) {
	res := ConfigurationCheckResult{}
	res.AddIssue("This is an issue")

	assert.Equal(t, 1, len(res.Issues), "Expected one issue in the result")
}
