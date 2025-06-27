package runtimeconfig

type CommonConfig struct {
	ConfigVersion int `yaml:"configVersion"`

	Amqp         AmqpConfig
	Connectors   []*ConnectorConfigWrapper
	Database     DatabaseConfig
	Nanoservices []NanoserviceConfig
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
