package hooks

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

// HTTPClient is a reusable HTTP client for webhook calls
var httpClient *http.Client

func init() {
	// Create HTTP client with timeout and skip TLS certificate validation
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip certificate validation for webhooks
		},
	}
	httpClient = &http.Client{
		Timeout:   60 * time.Second, // 1 minute timeout for webhook calls
		Transport: transport,
	}
}

// ExecuteHooks calls all enabled hooks for a given message
func ExecuteHooks(hooks []runtimeconfig.IncomingMessageHook, msg *msgs.IncomingMessage, logger log.FieldLogger) {
	if len(hooks) == 0 {
		return
	}

	// Prepare payload - send the full IncomingMessage structure as JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		if logger != nil {
			logger.Errorf("Failed to marshal IncomingMessage to JSON for hooks: %v", err)
		} else {
			log.Errorf("Failed to marshal IncomingMessage to JSON for hooks: %v", err)
		}
		return
	}

	// Execute each enabled hook
	for i, hook := range hooks {
		if !hook.Enabled || hook.URL == "" {
			continue
		}

		go executeHook(hook.URL, jsonData, i, msg, logger)
	}
}

func executeHook(url string, jsonData []byte, index int, msg *msgs.IncomingMessage, logger log.FieldLogger) {
	// Log the full URL being called
	if logger != nil {
		logger.Infof("Hook %d: Calling webhook URL: %s", index, url)
	} else {
		log.Infof("Hook %d: Calling webhook URL: %s", index, url)
	}

	// Create HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // 1 minute timeout
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to create webhook request: %v", err)
		if logger != nil {
			logger.Errorf("Hook %d: Failed to create HTTP request to %s: %v", index, url, err)
		} else {
			log.Errorf("Hook %d: Failed to create HTTP request to %s: %v", index, url, err)
		}

		// Send error notification to user
		sendErrorNotification(msg, errorMsg, logger)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("Webhook call failed: %v", err)
		if logger != nil {
			logger.Errorf("Hook %d: Failed to send message to webhook %s: %v", index, url, err)
		} else {
			log.Errorf("Hook %d: Failed to send message to webhook %s: %v", index, url, err)
		}

		// Send error notification to user
		sendErrorNotification(msg, errorMsg, logger)
		return
	}
	defer resp.Body.Close()

	// Read response body for logging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Warnf("Hook %d: Failed to read response body from %s: %v", index, url, err)
		} else {
			log.Warnf("Hook %d: Failed to read response body from %s: %v", index, url, err)
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if logger != nil {
			logger.Debugf("Hook %d: Successfully sent message to webhook %s (status: %d)", index, url, resp.StatusCode)
			if len(body) > 0 {
				logger.Debugf("Hook %d: Webhook response: %s", index, string(body))
			}
		}

		// Extract "output" field from webhook response and emit OutgoingMessage
		if len(body) > 0 {
			var responseData map[string]interface{}
			if err := json.Unmarshal(body, &responseData); err != nil {
				if logger != nil {
					logger.Warnf("Hook %d: Failed to parse webhook response JSON: %v", index, err)
				} else {
					log.Warnf("Hook %d: Failed to parse webhook response JSON: %v", index, err)
				}
			} else {
				if output, ok := responseData["output"]; ok {
					outputStr := ""
					switch v := output.(type) {
					case string:
						outputStr = v
					case float64:
						outputStr = fmt.Sprintf("%g", v)
					case bool:
						if v {
							outputStr = "true"
						} else {
							outputStr = "false"
						}
					default:
						// For other types, marshal back to JSON string
						if outputBytes, err := json.Marshal(v); err == nil {
							outputStr = string(outputBytes)
						} else {
							outputStr = ""
						}
					}

					if outputStr != "" {
						outgoingMsg := &msgs.OutgoingMessage{
							Content:            outputStr,
							Channel:            msg.Channel,
							Protocol:           msg.Protocol,
							IncommingMessageId: msg.MessageId,
							Identity:           msg.Identity, // Include bot identity to route to correct bot instance
						}

						routingKey := amqp.GetOutgoingMessageRoutingKey(msg.Protocol, msg.Identity)
						amqp.PublishPbWithRoutingKey(outgoingMsg, routingKey)

						if logger != nil {
							logger.Debugf("Hook %d: Emitted OutgoingMessage with output: %s", index, outputStr)
						} else {
							log.Debugf("Hook %d: Emitted OutgoingMessage with output: %s", index, outputStr)
						}
					}
				}
			}
		}
	} else {
		errorMsg := fmt.Sprintf("Webhook returned error status %d", resp.StatusCode)
		if len(body) > 0 {
			errorMsg = fmt.Sprintf("Webhook returned error status %d: %s", resp.StatusCode, string(body))
		}

		if logger != nil {
			logger.Errorf("Hook %d: Webhook %s returned error status: %d, response: %s", index, url, resp.StatusCode, string(body))
		} else {
			log.Errorf("Hook %d: Webhook %s returned error status: %d, response: %s", index, url, resp.StatusCode, string(body))
		}

		// Send error notification to user
		sendErrorNotification(msg, errorMsg, logger)
	}
}

// sendErrorNotification sends an OutgoingMessage to notify the user of a webhook failure
func sendErrorNotification(msg *msgs.IncomingMessage, errorMsg string, logger log.FieldLogger) {
	notificationMsg := fmt.Sprintf("⚠️ Webhook error: %s", errorMsg)

	outgoingMsg := &msgs.OutgoingMessage{
		Content:            notificationMsg,
		Channel:            msg.Channel,
		Protocol:           msg.Protocol,
		IncommingMessageId: msg.MessageId,
		Identity:           msg.Identity,
	}

	routingKey := amqp.GetOutgoingMessageRoutingKey(msg.Protocol, msg.Identity)
	amqp.PublishPbWithRoutingKey(outgoingMsg, routingKey)

	if logger != nil {
		logger.Debugf("Sent error notification to user: %s", notificationMsg)
	} else {
		log.Debugf("Sent error notification to user: %s", notificationMsg)
	}
}

// GetHooksForConnector returns the hooks configured for a specific connector
// It first tries to load from database, then falls back to config for backward compatibility
func GetHooksForConnector(connectorType string, identity string, dbInstance *db.DB) []runtimeconfig.IncomingMessageHook {
	// Try to load from database first
	if dbInstance != nil {
		dbHooks, err := dbInstance.SelectWebhookHooks(connectorType, identity)
		if err == nil && len(dbHooks) > 0 {
			hooks := make([]runtimeconfig.IncomingMessageHook, 0, len(dbHooks))
			for _, hook := range dbHooks {
				hooks = append(hooks, runtimeconfig.IncomingMessageHook{
					URL:     hook.URL,
					Enabled: hook.Enabled,
				})
			}
			return hooks
		}
	}

	// Fall back to config for backward compatibility
	cfg := runtimeconfig.Get()

	for _, wrapper := range cfg.Connectors {
		if wrapper.ConnectorType != connectorType {
			continue
		}

		// Check if this is the right connector instance
		if tgConfig, ok := wrapper.ConnectorConfig.(*runtimeconfig.TelegramConfig); ok {
			// For Telegram, match by name if provided, or by identity
			configName := tgConfig.Name
			if configName != "" && configName == identity {
				return tgConfig.IncomingMessageHooks
			}
			// If no name is set, return hooks for the first matching connector type
			// (This is a limitation - we'd need better instance identification)
			if configName == "" {
				return tgConfig.IncomingMessageHooks
			}
		}
	}

	return []runtimeconfig.IncomingMessageHook{}
}
