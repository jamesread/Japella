package connector

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationCheckResult(t *testing.T) {
	res := ConfigurationCheckResult{}
	res.AddIssue("This is an issue")

	assert.Equal(t, 1, len(res.Issues), "Expected one issue in the result")
}
