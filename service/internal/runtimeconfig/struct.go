package runtimeconfig

type CommonConfig struct {
	ConfigVersion int `yaml:"configVersion"`

	Amqp       AmqpConfig
	Connectors []*ConnectorConfigWrapper
	Database   DatabaseConfig
}

type ConnectorConfig interface {
}

type ConnectorConfigWrapper struct {
	ConnectorConfig ConnectorConfig
	ConnectorType   string
	Enabled         bool
}

type DiscordConfig struct {
	AppId     string
	PublicKey string
	Token     string
}

type TelegramConfig struct {
	Token string
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
	Enabled  bool
	Host     string
	User     string
	Password string
	Database string
}

type MastodonConfig struct {
	Register     bool
	AppId        string
	ClientId     string
	ClientSecret string
	Website      string
	Token        string

	Inert bool
}

type WhatsAppConfig struct {
	AccessToken       string
	PhoneNumberID     string
	BusinessAccountID string
}
