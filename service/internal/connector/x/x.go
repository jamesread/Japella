package x

import (
	"github.com/jamesread/japella/internal/connector"
)

type XConnector struct {
	connector.BaseConnector
	connector.ConnectorWithWall
}

func (x *XConnector) StartWithConfig(config any) {
}

func (x *XConnector) GetIdentity() string {
	return "?"
}

func (x *XConnector) GetProtocol() string {
	return "x"
}

func (x *XConnector) PostToWall(message string) error {
	return nil
}

func (x *XConnector) GetIcon() string {
	return "mdi:twitter"
}
