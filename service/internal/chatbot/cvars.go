package chatbot

import "fmt"

const CvarCategory = "Chat Bots"

func TelegramBotTokenKey(botID string) string {
	return fmt.Sprintf("bot.telegram.%s.bottoken", botID)
}

func DiscordTokenKey(botID string) string {
	return fmt.Sprintf("bot.discord.%s.token", botID)
}

func DiscordAppIDKey(botID string) string {
	return fmt.Sprintf("bot.discord.%s.app_id", botID)
}

func DiscordPublicKeyKey(botID string) string {
	return fmt.Sprintf("bot.discord.%s.public_key", botID)
}

func CvarKeysForInstance(protocol, botID string) []string {
	switch protocol {
	case "telegram":
		return []string{TelegramBotTokenKey(botID)}
	case "discord":
		return []string{
			DiscordTokenKey(botID),
			DiscordAppIDKey(botID),
			DiscordPublicKeyKey(botID),
		}
	default:
		return nil
	}
}

func IsBotCvarKey(key string) bool {
	return len(key) > 4 && key[:4] == "bot."
}
