package connectorcontroller

import (
	"github.com/google/uuid"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connector/discord"
	"github.com/jamesread/japella/internal/connector/mastodon"
	"github.com/jamesread/japella/internal/connector/telegram"
	"github.com/jamesread/japella/internal/connector/x"
	"github.com/jamesread/japella/internal/connector/bluesky"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

type ConnectionController struct {
	controllers map[string]connector.BaseConnector
}

func New() *ConnectionController {
	cc := &ConnectionController{
		controllers: map[string]connector.BaseConnector{},
	}

	for _, wrapper := range runtimeconfig.Get().Connectors {
		if wrapper.Enabled {
			cc.startControllerFromConfig(wrapper)
		} else {
			log.Warnf("Connector %s is disabled in configuration", wrapper.ConnectorType)
		}
	}

	return cc
}

func (cc *ConnectionController) Get(key string) connector.BaseConnector {
	controller, exists := cc.controllers[key]

	if !exists {
		log.Errorf("Controller not found: %s", key)
		return nil
	}

	return controller
}

func (cc *ConnectionController) GetServices() map[string]connector.BaseConnector {
	return cc.controllers
}

func (cc *ConnectionController) GetKeys() []string {
	keys := make([]string, 0)

	log.Infof("Registered controllers: %v", cc)

	for k := range cc.controllers {
		keys = append(keys, k)
	}

	return keys
}

func (cc *ConnectionController) startControllerFromConfig(wrapper *runtimeconfig.ConnectorConfigWrapper) {
	log.Infof("Registering controller, type: %v", wrapper.ConnectorType)

	switch wrapper.ConnectorType {
	case "telegram":
		cc.setupConnector(&telegram.TelegramConnector{}, wrapper.ConnectorConfig)
	case "discord":
		cc.setupConnector(&discord.DiscordConnector{}, wrapper.ConnectorConfig)
	case "mastodon":
		cc.setupConnector(&mastodon.MastodonConnector{}, wrapper.ConnectorConfig)
	case "x":
		cc.setupConnector(&x.XConnector{}, wrapper.ConnectorConfig)
	case "bluesky":
		cc.setupConnector(&bluesky.BlueskyConnector{}, wrapper.ConnectorConfig)
	default:
		log.Errorf("Unknown controller type: " + wrapper.ConnectorType)
	}
}

func (cc *ConnectionController) setupConnector(c connector.BaseConnector, config any) {
	go c.StartWithConfig(config)

	controllerID := uuid.New().String()

	log.Infof("Setting up connector %v with id %v", c.GetProtocol(), controllerID)
	cc.RegisterController(controllerID, c)
}

func (cc *ConnectionController) RegisterController(name string, controller connector.BaseConnector) {
	if _, exists := cc.controllers[name]; exists {
		panic("Connector registered: " + name)
	}

	cc.controllers[name] = controller
}
