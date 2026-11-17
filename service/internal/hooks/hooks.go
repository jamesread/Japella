package hooks

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

const hookTimeout = 60 * time.Second

var (
	verifyingClient = &http.Client{Timeout: hookTimeout}

	// Only for hooks whose owner opted out of TLS verification, e.g. an
	// endpoint with a self-signed certificate.
	insecureClient = &http.Client{
		Timeout: hookTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // per-hook opt-out, off by default
		},
	}
)

// publishOutgoing sends a reply to the chat the message came from. Tests replace it.
var publishOutgoing = func(m *msgs.OutgoingMessage) {
	amqp.PublishPbWithRoutingKey(m, amqp.GetOutgoingMessageRoutingKey(m.Protocol, m.Identity))
}

func clientFor(skipTLSVerify bool) *http.Client {
	if skipTLSVerify {
		return insecureClient
	}
	return verifyingClient
}

// ExecuteHooks calls all enabled hooks for a given message
func ExecuteHooks(hooks []runtimeconfig.IncomingMessageHook, msg *msgs.IncomingMessage, logger log.FieldLogger) {
	if len(hooks) == 0 {
		return
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		orStandardLogger(logger).Errorf("Failed to marshal IncomingMessage to JSON for hooks: %v", err)
		return
	}
	startEnabledHooks(hooks, jsonData, msg, logger)
}

func startEnabledHooks(hooks []runtimeconfig.IncomingMessageHook, jsonData []byte, msg *msgs.IncomingMessage, logger log.FieldLogger) {
	for i, hook := range hooks {
		if !hook.Enabled || hook.URL == "" {
			continue
		}
		go executeHook(hook, jsonData, i, msg, logger)
	}
}

func executeHook(hook runtimeconfig.IncomingMessageHook, jsonData []byte, index int, msg *msgs.IncomingMessage, logger log.FieldLogger) {
	logger = orStandardLogger(logger)
	logger.Infof("Hook %d: Calling webhook URL: %s", index, hook.URL)

	status, body, failure := callHook(hook, jsonData, index, logger)
	switch {
	case failure != "":
		sendErrorNotification(msg, failure, logger)
	case status < 200 || status >= 300:
		reportHookErrorStatus(msg, hook.URL, index, status, body, logger)
	default:
		logger.Debugf("Hook %d: Successfully sent message to webhook %s (status: %d)", index, hook.URL, status)
		emitHookOutput(msg, body, index, logger)
	}
}

func orStandardLogger(logger log.FieldLogger) log.FieldLogger {
	if logger == nil {
		return log.StandardLogger()
	}
	return logger
}

// callHook posts the payload. When the call itself fails, failure holds the
// message to send back to the chat.
func callHook(hook runtimeconfig.IncomingMessageHook, jsonData []byte, index int, logger log.FieldLogger) (status int, body []byte, failure string) {
	ctx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Errorf("Hook %d: Failed to create HTTP request to %s: %v", index, hook.URL, err)
		return 0, nil, fmt.Sprintf("Failed to create webhook request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := clientFor(hook.SkipTLSVerify).Do(req)
	if err != nil {
		logger.Errorf("Hook %d: Failed to send message to webhook %s: %v", index, hook.URL, err)
		return 0, nil, fmt.Sprintf("Webhook call failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf("Hook %d: Failed to read response body from %s: %v", index, hook.URL, err)
	}
	return resp.StatusCode, body, ""
}

func reportHookErrorStatus(msg *msgs.IncomingMessage, url string, index, status int, body []byte, logger log.FieldLogger) {
	errorMsg := fmt.Sprintf("Webhook returned error status %d", status)
	if len(body) > 0 {
		errorMsg = fmt.Sprintf("Webhook returned error status %d: %s", status, string(body))
	}
	logger.Errorf("Hook %d: Webhook %s returned error status: %d, response: %s", index, url, status, string(body))
	sendErrorNotification(msg, errorMsg, logger)
}

// emitHookOutput sends the response's "output" field, if any, back to the chat.
func emitHookOutput(msg *msgs.IncomingMessage, body []byte, index int, logger log.FieldLogger) {
	if len(body) == 0 {
		return
	}
	logger.Debugf("Hook %d: Webhook response: %s", index, string(body))

	output, err := hookOutput(body)
	if err != nil {
		logger.Warnf("Hook %d: Failed to parse webhook response JSON: %v", index, err)
		return
	}
	if output == "" {
		return
	}

	publishOutgoing(&msgs.OutgoingMessage{
		Content:            output,
		Channel:            msg.Channel,
		Protocol:           msg.Protocol,
		IncommingMessageId: msg.MessageId,
		Identity:           msg.Identity, // Include bot identity to route to correct bot instance
		ConversationKey:    db.BuildConversationKey(msg.Channel, msg.Author),
	})
	logger.Debugf("Hook %d: Emitted OutgoingMessage with output: %s", index, output)
}

// hookOutput returns the response's "output" field as text, or "" if absent.
// A JSON null becomes "null".
func hookOutput(body []byte) (string, error) {
	var responseData map[string]any
	if err := json.Unmarshal(body, &responseData); err != nil {
		return "", err
	}
	output, ok := responseData["output"]
	if !ok {
		return "", nil
	}
	return outputText(output), nil
}

func outputText(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return jsonText(v)
	}
}

func jsonText(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func sendErrorNotification(msg *msgs.IncomingMessage, errorMsg string, logger log.FieldLogger) {
	notificationMsg := fmt.Sprintf("⚠️ Webhook error: %s", errorMsg)

	outgoingMsg := &msgs.OutgoingMessage{
		Content:            notificationMsg,
		Channel:            msg.Channel,
		Protocol:           msg.Protocol,
		IncommingMessageId: msg.MessageId,
		Identity:           msg.Identity,
		ConversationKey:    db.BuildConversationKey(msg.Channel, msg.Author),
	}

	publishOutgoing(outgoingMsg)

	if logger != nil {
		logger.Debugf("Sent error notification to user: %s", notificationMsg)
	} else {
		log.Debugf("Sent error notification to user: %s", notificationMsg)
	}
}

// FromDB converts stored hooks to the form ExecuteHooks takes.
func FromDB(dbHooks []*db.WebhookHook) []runtimeconfig.IncomingMessageHook {
	hooks := make([]runtimeconfig.IncomingMessageHook, 0, len(dbHooks))
	for _, hook := range dbHooks {
		hooks = append(hooks, runtimeconfig.IncomingMessageHook{
			URL:           hook.URL,
			Enabled:       hook.Enabled,
			SkipTLSVerify: hook.SkipTLSVerify,
		})
	}
	return hooks
}

// GetHooksForConnector returns the hooks configured for a specific connector
// It first tries to load from database, then falls back to config for backward compatibility
func GetHooksForConnector(connectorType string, identity string, dbInstance *db.DB) []runtimeconfig.IncomingMessageHook {
	if hooks, ok := hooksFromDB(connectorType, identity, dbInstance); ok {
		return hooks
	}
	return hooksFromConfig(connectorType, identity)
}

func hooksFromDB(connectorType, identity string, dbInstance *db.DB) ([]runtimeconfig.IncomingMessageHook, bool) {
	if dbInstance == nil {
		return nil, false
	}
	dbHooks, err := dbInstance.SelectWebhookHooks(connectorType, identity)
	if err != nil || len(dbHooks) == 0 {
		return nil, false
	}
	return FromDB(dbHooks), true
}

func hooksFromConfig(connectorType, identity string) []runtimeconfig.IncomingMessageHook {
	for _, wrapper := range runtimeconfig.Get().Connectors {
		if wrapper.ConnectorType != connectorType {
			continue
		}
		if hooks, ok := telegramHooksForIdentity(wrapper.ConnectorConfig, identity); ok {
			return hooks
		}
	}
	return []runtimeconfig.IncomingMessageHook{}
}

func telegramHooksForIdentity(cfg any, identity string) ([]runtimeconfig.IncomingMessageHook, bool) {
	tgConfig, ok := cfg.(*runtimeconfig.TelegramConfig)
	if !ok {
		return nil, false
	}
	// Match by name when set; otherwise first matching connector type (legacy).
	if tgConfig.Name != "" && tgConfig.Name != identity {
		return nil, false
	}
	return tgConfig.IncomingMessageHooks, true
}
