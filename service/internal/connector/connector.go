package connector

type BaseConnector interface {
	StartWithConfig(config any)
	GetIdentity() string
	GetProtocol() string
	GetIcon() string
}

type ConnectorWithChannels interface {
	PostToChannel(channelId string, message string)
}

type ConnectorWithWall interface {
	PostToWall(message string) error
}
