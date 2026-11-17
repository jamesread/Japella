package api

import (
	"testing"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connectorcontroller"
	"github.com/stretchr/testify/assert"
)

func TestServiceStatus(t *testing.T) {
	existing := &controlv1.StatusMessage{Type: "note", Message: "existing"}

	// No database: ReadinessClientMessage explains why.
	s := &ControlApi{statusMessages: []*controlv1.StatusMessage{existing}}
	status, messages := s.serviceStatus(false)
	assert.Equal(t, "ERROR", status)
	assert.Equal(t, []*controlv1.StatusMessage{existing, {Type: "error", Message: "Service is not available."}}, messages)
	assert.Len(t, s.statusMessages, 1, "the service's own messages are not modified")

	status, messages = s.serviceStatus(true)
	assert.Equal(t, "STARTING", status)
	assert.Equal(t, []*controlv1.StatusMessage{existing, {Type: "warning", Message: "Service is starting"}}, messages)

	s.cc = &connectorcontroller.ConnectionController{}
	status, messages = s.serviceStatus(true)
	assert.Equal(t, "OK!", status)
	assert.Equal(t, []*controlv1.StatusMessage{existing}, messages)
}
