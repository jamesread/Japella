package runtimeconfig

type CommonConfig struct {
	ConfigVersion int `yaml:"configVersion"`

	Amqp              AmqpConfig
	Connectors        []*ConnectorConfigWrapper
	Database          DatabaseConfig
	Nanoservices      []NanoserviceConfig
	TLS               TLSConfig
	ListenAddress     string `yaml:"listenAddress"`
	TelegramDebugChatId string `yaml:"telegramDebugChatId"` // Global debug chat ID for Telegram bots to send startup messages
}

type ConnectorConfig interface {
}

type NanoserviceConfig struct {
	Name string
}

type ConnectorConfigWrapper struct {
	ConnectorConfig ConnectorConfig
	ConnectorType   string
	Enabled         bool
}

type BlueskyConfig struct {
}

type DiscordConfig struct {
	AppId                string
	PublicKey            string
	Token                string
	IncomingMessageHooks []IncomingMessageHook `yaml:"incomingMessageHooks"` // Webhooks to call when messages are received
}

type IncomingMessageHook struct {
	URL     string `yaml:"url"`     // Webhook URL to call
	Enabled bool   `yaml:"enabled"` // Whether this hook is enabled
}

type TelegramConfig struct {
	Token                string                `yaml:"token"`
	Name                 string                `yaml:"name"`
	IncomingMessageHooks []IncomingMessageHook `yaml:"incomingMessageHooks"` // Webhooks to call when messages are received
}

type AmqpConfig struct {
	Enabled  bool
	Host     string
	User     string
	Pass     string
	Port     int
	Exchange string
}

type DatabaseConfig struct {
	Enabled bool
	Host    string
	User    string
	Pass    string
	Port    int
	Name    string
}

type WhatsAppConfig struct {
	AccessToken       string
	PhoneNumberID     string
	BusinessAccountID string
}

type TLSConfig struct {
	CrtPath string `yaml:"crtPath"`
	KeyPath string `yaml:"keyPath"`
}
