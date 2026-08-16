package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

const EventApprovalRequested = "approval.requested"
const EventPostCompleted = "post.completed"
const EventPostError = "post.error"

var SupportedEvents = []string{EventApprovalRequested, EventPostCompleted, EventPostError}

func NormalizeEvent(event string) (string, error) {
	event = strings.TrimSpace(event)
	for _, e := range SupportedEvents {
		if event == e {
			return event, nil
		}
	}
	return "", fmt.Errorf("unsupported webhook event")
}

func NormalizeEvents(events []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(events))
	for _, raw := range events {
		e, err := NormalizeEvent(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out, nil
}

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("webhook URL must be non-empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("webhook URL is not valid")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("webhook URL scheme must be http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("webhook URL host is required")
	}
	return raw, nil
}

func Signature(payloadJSON, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadJSON))
	return hex.EncodeToString(mac.Sum(nil))
}
