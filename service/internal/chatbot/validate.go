package chatbot

import (
	"fmt"
	"regexp"
	"strings"
)

var botIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,30}$`)

var reservedBotIDs = map[string]bool{
	"_": true, "yaml": true, "new": true, "create": true,
}

func ValidateBotID(botID string) error {
	if botID == "" {
		return fmt.Errorf("bot id is required")
	}
	if !botIDPattern.MatchString(botID) {
		return fmt.Errorf("bot id must be 2-31 lowercase letters, digits, underscores, or hyphens, starting with a letter")
	}
	if reservedBotIDs[botID] {
		return fmt.Errorf("bot id %q is reserved", botID)
	}
	return nil
}

func ValidateProtocol(protocol string) error {
	switch strings.ToLower(protocol) {
	case "telegram", "discord":
		return nil
	default:
		return fmt.Errorf("unsupported chat bot protocol: %s", protocol)
	}
}

func ControllerKey(protocol, botID string) string {
	return protocol + "-" + botID
}
