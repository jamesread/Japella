package connectorcontroller

import (
	"fmt"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connector/bluesky"
	"github.com/jamesread/japella/internal/connector/discord"
	"github.com/jamesread/japella/internal/connector/facebook"
	"github.com/jamesread/japella/internal/connector/instagram"
	"github.com/jamesread/japella/internal/connector/mastodon"
	"github.com/jamesread/japella/internal/connector/telegram"
	"github.com/jamesread/japella/internal/connector/x"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

type ConnectionController struct {
	controllers map[string]connector.BaseConnector
	db          *db.DB
}

func New(dbc *db.DB) *ConnectionController {
	cc := &ConnectionController{
		controllers: map[string]connector.BaseConnector{},
		db:          dbc,
	}

	for _, wrapper := range runtimeconfig.Get().Connectors {
		if wrapper.Enabled {
			cc.startControllerFromConfig(wrapper)
		} else {
			log.Warnf("Connector %s is disabled in configuration", wrapper.ConnectorType)
		}
	}

	cc.setupConnector(&mastodon.MastodonConnector{}, nil)
	cc.setupConnector(&x.XConnector{}, nil)
	cc.setupConnector(&bluesky.BlueskyConnector{}, nil)
	cc.setupConnector(&facebook.FacebookConnector{}, nil)
	cc.setupConnector(&instagram.InstagramConnector{}, nil)

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

// GetAllAvailableConnectorTypes returns a list of all connector types that the system knows about
func GetAllAvailableConnectorTypes() []string {
	return []string{
		"telegram",
		"discord",
		"mastodon",
		"x",
		"bluesky",
		"facebook",
		"instagram",
		"whatsapp",
	}
}

// GetUnregisteredConnectors returns connector info for connectors that exist but aren't started
func (cc *ConnectionController) GetUnregisteredConnectors() []*connector.UnregisteredConnector {
	available := GetAllAvailableConnectorTypes()
	startedTypes := make(map[string]bool)

	// Check if any connector of each type is started (keys may be "telegram", "telegram-1", "telegram-MyBot", etc.)
	for k := range cc.controllers {
		// Extract the base protocol from the key (everything before the first dash)
		for _, connectorType := range available {
			if k == connectorType || (len(k) > len(connectorType) && k[:len(connectorType)+1] == connectorType+"-") {
				startedTypes[connectorType] = true
				break
			}
		}
	}

	unregistered := make([]*connector.UnregisteredConnector, 0)

	// Icon map for connectors that might not fully implement the interface
	iconMap := map[string]string{
		"telegram":  "mdi:telegram",
		"discord":   "mdi:discord",
		"mastodon":  "mdi:mastodon",
		"x":         "mdi:twitter",
		"bluesky":   "mdi:bluesky",
		"facebook":  "mdi:facebook",
		"instagram": "mdi:instagram",
		"whatsapp":  "mdi:whatsapp",
	}

	for _, connectorType := range available {
		if !startedTypes[connectorType] {
			// Create a temporary instance to get icon and other info
			conn := cc.createConnectorInstance(connectorType)
			icon := iconMap[connectorType] // Default icon
			if conn != nil {
				icon = conn.GetIcon()
			}

			unregistered = append(unregistered, &connector.UnregisteredConnector{
				Protocol: connectorType,
				Icon:     icon,
				Name:     connectorType,
			})
		}
	}

	return unregistered
}

// createConnectorInstance creates a temporary connector instance to get metadata
func (cc *ConnectionController) createConnectorInstance(connectorType string) connector.BaseConnector {
	switch connectorType {
	case "telegram":
		return &telegram.TelegramConnector{}
	case "discord":
		return &discord.DiscordConnector{}
	case "mastodon":
		return &mastodon.MastodonConnector{}
	case "x":
		return &x.XConnector{}
	case "bluesky":
		return &bluesky.BlueskyConnector{}
	case "facebook":
		return &facebook.FacebookConnector{}
	case "instagram":
		return &instagram.InstagramConnector{}
	case "whatsapp":
		// WhatsApp connector doesn't fully implement BaseConnector, return nil
		// Icon will be handled via iconMap in GetUnregisteredConnectors
		return nil
	default:
		return nil
	}
}

func (cc *ConnectionController) startControllerFromConfig(wrapper *runtimeconfig.ConnectorConfigWrapper) {
	log.Infof("Registering controller, type: %v", wrapper.ConnectorType)

	// Generate a unique key for this connector instance
	// Use protocol + name (if available) + index to ensure uniqueness
	var instanceKey string
	var connectorInstance connector.BaseConnector

	switch wrapper.ConnectorType {
	case "telegram":
		connectorInstance = &telegram.TelegramConnector{}
		// Try to get name from config for unique key
		if tgConfig, ok := wrapper.ConnectorConfig.(*runtimeconfig.TelegramConfig); ok && tgConfig.Name != "" {
			instanceKey = "telegram-" + tgConfig.Name
		} else {
			// Use index-based key if no name
			instanceKey = cc.generateUniqueKey("telegram")
		}
	case "discord":
		connectorInstance = &discord.DiscordConnector{}
		instanceKey = cc.generateUniqueKey("discord")
	case "bluesky":
		connectorInstance = &bluesky.BlueskyConnector{}
		instanceKey = cc.generateUniqueKey("bluesky")
	case "facebook":
		connectorInstance = &facebook.FacebookConnector{}
		instanceKey = cc.generateUniqueKey("facebook")
	case "instagram":
		connectorInstance = &instagram.InstagramConnector{}
		instanceKey = cc.generateUniqueKey("instagram")
	default:
		log.Errorf("Unknown controller type: " + wrapper.ConnectorType)
		return
	}

	if connectorInstance != nil {
		cc.setupConnectorWithKey(connectorInstance, wrapper.ConnectorConfig, instanceKey)
	}
}

func (cc *ConnectionController) setupConnector(c connector.BaseConnector, config any) {
	name := c.GetProtocol()
	cc.setupConnectorWithKey(c, config, name)
}

func (cc *ConnectionController) setupConnectorWithKey(c connector.BaseConnector, config any, key string) {
	if _, exists := cc.controllers[key]; exists {
		log.Warnf("Connector with key %s already exists, skipping", key)
		return
	}

	log.Infof("Setting up connector: %v (key: %s)", c.GetProtocol(), key)

	startupConfiguration := &connector.ControllerStartupConfiguration{
		Config: config,
		DB:     cc.db,
	}

	go c.SetStartupConfiguration(startupConfiguration)

	configProvider, ok := c.(connector.ConfigProvider)

	if ok {
		cvars := configProvider.GetCvars()

		for _, cvar := range cvars {
			if err := cc.db.InsertCvarIfNotExists(cvar); err != nil {
				log.Errorf("Error creating cvar %s: %v", cvar.KeyName, err)
			}
		}
	}

	c.Start()

	cc.controllers[key] = c
}

// generateUniqueKey generates a unique key for a connector instance
func (cc *ConnectionController) generateUniqueKey(protocol string) string {
	baseKey := protocol
	counter := 0

	for {
		key := baseKey
		if counter > 0 {
			key = fmt.Sprintf("%s-%d", baseKey, counter)
		}

		if _, exists := cc.controllers[key]; !exists {
			return key
		}

		counter++
	}
}
