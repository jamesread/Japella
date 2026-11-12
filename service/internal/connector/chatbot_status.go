package connector

func ChatBotStatusMessage(isRunning bool, errorMessage, statusMessage string) string {
	switch {
	case isRunning:
		return "Bot is running and connected"
	case errorMessage != "":
		return errorMessage
	case statusMessage != "":
		return statusMessage
	default:
		return "Stopped (not connected)"
	}
}
