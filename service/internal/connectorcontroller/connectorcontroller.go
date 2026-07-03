package connectorcontroller

import (
	"fmt"
	"sync"

	"github.com/jamesread/japella/internal/chatbot"
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

var oauthConnectorTypes = map[string]bool{
	"mastodon": true, "x": true, "bluesky": true, "facebook": true, "instagram": true,
}

var chatBotProtocols = map[string]bool{
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
		if chatBotProtocols[wrapper.ConnectorType] {
			log.Infof("Skipping YAML connector %s: chat bots are configured via the Web UI", wrapper.ConnectorType)
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

	cc.loadChatBotsFromDB()

	return cc
}

func (cc *ConnectionController) loadChatBotsFromDB() {
	instances, err := cc.db.SelectChatBotInstances()
	if err != nil {
		log.Errorf("Failed to load chat bot instances: %v", err)
		return
	}
	for _, inst := range instances {
		if err := cc.registerChatBotInstanceLocked(inst.Protocol, inst.BotID, inst.DisplayName); err != nil {
			log.Errorf("Failed to register chat bot %s/%s: %v", inst.Protocol, inst.BotID, err)
		}
	}
}

func (cc *ConnectionController) RegisterChatBotInstance(protocol, botID, displayName string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.registerChatBotInstanceLocked(protocol, botID, displayName)
}

func (cc *ConnectionController) registerChatBotInstanceLocked(protocol, botID, displayName string) error {
	key := chatbot.ControllerKey(protocol, botID)
	if _, exists := cc.controllers[key]; exists {
		return fmt.Errorf("chat bot already registered: %s", key)
	}

	var instance connector.BaseConnector
	switch protocol {
	case "telegram":
		instance = &telegram.TelegramConnector{}
	case "discord":
		instance = &discord.DiscordConnector{}
	default:
		return fmt.Errorf("unsupported chat bot protocol: %s", protocol)
	}

	cfg := &connector.ChatBotStartupConfig{
		Protocol:    protocol,
		BotID:       botID,
		DisplayName: displayName,
	}
	cc.setupConnectorWithKey(instance, cfg, key)
	return nil
}

func (cc *ConnectionController) UnregisterChatBotInstance(protocol, botID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	key := chatbot.ControllerKey(protocol, botID)
	svc, exists := cc.controllers[key]
	if !exists {
		return fmt.Errorf("chat bot not registered: %s", key)
	}

	if ctrl, ok := svc.(connector.ConnectorWithChatBotControl); ok {
		_ = ctrl.StopChatBot()
	}

	delete(cc.controllers, key)
	log.Infof("Unregistered chat bot: %s", key)
	return nil
}

func (cc *ConnectionController) GetChatBotControl(protocol, botID string) (connector.ConnectorWithChatBotControl, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	key := chatbot.ControllerKey(protocol, botID)
	svc, exists := cc.controllers[key]
	if !exists {
		return nil, fmt.Errorf("chat bot not found: %s", key)
	}
	ctrl, ok := svc.(connector.ConnectorWithChatBotControl)
	if !ok {
		return nil, fmt.Errorf("connector does not support chat bot control: %s", protocol)
	}
	return ctrl, nil
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

func (cc *ConnectionController) GetUnregisteredConnectors() []*connector.UnregisteredConnector {
	cc.mu.RLock()
	available := GetAllAvailableConnectorTypes()
	startedTypes := make(map[string]bool)

	for k := range cc.controllers {
		for _, connectorType := range available {
			if k == connectorType || (len(k) > len(connectorType) && k[:len(connectorType)+1] == connectorType+"-") {
				startedTypes[connectorType] = true
				break
			}
		}
	}
	cc.mu.RUnlock()

	isPubliclyAccessible := cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)

	unregistered := make([]*connector.UnregisteredConnector, 0)

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
		if chatBotProtocols[connectorType] {
			continue
		}
		if !startedTypes[connectorType] {
			conn := cc.createConnectorInstance(connectorType)
			icon := iconMap[connectorType]
			if conn != nil {
				icon = conn.GetIcon()
			}

			reason := getNotStartedReason(connectorType, isPubliclyAccessible)

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

func getNotStartedReason(connectorType string, isPubliclyAccessible bool) string {
	if oauthConnectorTypes[connectorType] && !isPubliclyAccessible {
		return "Requires IsPubliclyAccessible to be enabled"
	}
	if connectorType == "whatsapp" {
		return "Not configured"
	}
	return "Not started"
}

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
	default:
		return nil
	}
}

func (cc *ConnectionController) startControllerFromConfig(wrapper *runtimeconfig.ConnectorConfigWrapper) {
	log.Infof("Registering controller, type: %v", wrapper.ConnectorType)

	var instanceKey string
	var connectorInstance connector.BaseConnector

	switch wrapper.ConnectorType {
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
		log.Warnf("Skipping YAML connector type %s (not supported via YAML)", wrapper.ConnectorType)
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

	c.SetStartupConfiguration(startupConfiguration)

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

func (cc *ConnectionController) RefreshConnectors() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	isPubliclyAccessible := cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)

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
