package connectorcontroller

import (
	"fmt"
	"sync"

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
	mu          sync.RWMutex
	db          *db.DB
}

// oauthConnectorTypes lists connector types that require OAuth and should only be started when IsPubliclyAccessible is true
var oauthConnectorTypes = map[string]bool{
	"mastodon": true, "x": true, "bluesky": true, "facebook": true, "instagram": true,
}

// yamlConnectorTypes lists connector types that require YAML configuration to be started
var yamlConnectorTypes = map[string]bool{
	"telegram": true, "discord": true,
}

func New(dbc *db.DB) *ConnectionController {
	cc := &ConnectionController{
		controllers: map[string]connector.BaseConnector{},
		db:          dbc,
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()

	isPubliclyAccessible := dbc.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)

	for _, wrapper := range runtimeconfig.Get().Connectors {
		if !wrapper.Enabled {
			log.Warnf("Connector %s is disabled in configuration", wrapper.ConnectorType)
			continue
		}
		if oauthConnectorTypes[wrapper.ConnectorType] && !isPubliclyAccessible {
			log.Infof("Skipping OAuth connector %s: IsPubliclyAccessible is false", wrapper.ConnectorType)
			continue
		}
		cc.startControllerFromConfig(wrapper)
	}

	if isPubliclyAccessible {
		cc.setupConnector(&mastodon.MastodonConnector{}, nil)
		cc.setupConnector(&x.XConnector{}, nil)
		cc.setupConnector(&bluesky.BlueskyConnector{}, nil)
		cc.setupConnector(&facebook.FacebookConnector{}, nil)
		cc.setupConnector(&instagram.InstagramConnector{}, nil)
	} else {
		log.Infof("OAuth connectors not started: IsPubliclyAccessible is false")
	}

	return cc
}

func (cc *ConnectionController) Get(key string) connector.BaseConnector {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	controller, exists := cc.controllers[key]

	if !exists {
		log.Errorf("Controller not found: %s", key)
		return nil
	}

	return controller
}

func (cc *ConnectionController) GetServices() map[string]connector.BaseConnector {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	services := make(map[string]connector.BaseConnector, len(cc.controllers))
	for k, v := range cc.controllers {
		services[k] = v
	}
	return services
}

func (cc *ConnectionController) GetKeys() []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
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
	cc.mu.RLock()
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
	cc.mu.RUnlock()

	isPubliclyAccessible := cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)
	configTypes := make(map[string]bool)
	for _, w := range runtimeconfig.Get().Connectors {
		configTypes[w.ConnectorType] = true
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

			reason := getNotStartedReason(connectorType, isPubliclyAccessible, configTypes)

			unregistered = append(unregistered, &connector.UnregisteredConnector{
				Protocol:         connectorType,
				Icon:             icon,
				Name:             connectorType,
				NotStartedReason: reason,
			})
		}
	}

	return unregistered
}

func getNotStartedReason(connectorType string, isPubliclyAccessible bool, configTypes map[string]bool) string {
	if oauthConnectorTypes[connectorType] && !isPubliclyAccessible {
		return "Requires IsPubliclyAccessible to be enabled"
	}
	if yamlConnectorTypes[connectorType] && !configTypes[connectorType] {
		return "Requires YAML configuration"
	}
	if yamlConnectorTypes[connectorType] && configTypes[connectorType] {
		return "Disabled in YAML configuration"
	}
	if connectorType == "whatsapp" {
		return "Not configured"
	}
	return "Not started"
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

// RefreshConnectors applies the current IsPubliclyAccessible setting by starting or stopping OAuth connectors.
// Call this after changing the IsPubliclyAccessible setting to apply it without restarting the server.
func (cc *ConnectionController) RefreshConnectors() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	isPubliclyAccessible := cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)

	// Remove OAuth connectors when IsPubliclyAccessible is false
	if !isPubliclyAccessible {
		for key := range cc.controllers {
			for oauthType := range oauthConnectorTypes {
				if key == oauthType || (len(key) > len(oauthType) && key[:len(oauthType)+1] == oauthType+"-") {
					delete(cc.controllers, key)
					log.Infof("Stopped OAuth connector: %s (IsPubliclyAccessible is false)", key)
					break
				}
			}
		}
		return
	}

	// Add OAuth connectors when IsPubliclyAccessible is true (if not already present)
	oauthDefaults := []struct {
		key string
		c   connector.BaseConnector
	}{
		{"mastodon", &mastodon.MastodonConnector{}},
		{"x", &x.XConnector{}},
		{"bluesky", &bluesky.BlueskyConnector{}},
		{"facebook", &facebook.FacebookConnector{}},
		{"instagram", &instagram.InstagramConnector{}},
	}
	for _, def := range oauthDefaults {
		if _, exists := cc.controllers[def.key]; !exists {
			cc.setupConnectorWithKey(def.c, nil, def.key)
			log.Infof("Started OAuth connector: %s (IsPubliclyAccessible is true)", def.key)
		}
	}
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
