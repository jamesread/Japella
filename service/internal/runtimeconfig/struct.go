package runtimeconfig

type CommonConfig struct {
	ConfigVersion int

	Amqp       *AmqpConfig
	Connectors *ConnectorConfig
	Database   *DatabaseConfig
}

type ConnectorConfig struct {
	Discord  *DiscordConfig
	Telegram *TelegramConfig
	Mastodon *MastodonConfig
	WhatsApp *WhatsAppConfig
}

type DiscordConfig struct {
	AppId     string
	PublicKey string
	Token     string
}

type TelegramConfig struct {
	BotToken string
}

type AmqpConfig struct {
	Host     string
	User     string
	Pass     string
	Port     int
	Exchange string
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Database string
}

type MastodonConfig struct {
	Register bool
	AppId    string
	ClientId string
	Website  string

	Inert bool
}

type WhatsAppConfig struct {
	AccessToken string
	PhoneNumberID string
	BusinessAccountID string
}
