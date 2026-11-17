package connectorcontroller

import (
	"fmt"
	"maps"
	"sync"

	"github.com/jamesread/japella/internal/chatbot"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connector/bluesky"
	"github.com/jamesread/japella/internal/connector/discord"
	"github.com/jamesread/japella/internal/connector/dummy"
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
	cc.startEnabledYAMLConnectors(isPubliclyAccessible)
	cc.startDefaultOAuthConnectors(isPubliclyAccessible)
	cc.loadChatBotsFromDB()

	// Always available for local/dev keyword-watch and feed testing.
	cc.setupConnector(&dummy.Connector{})

	return cc
}

func (cc *ConnectionController) startEnabledYAMLConnectors(isPubliclyAccessible bool) {
	for _, wrapper := range runtimeconfig.Get().Connectors {
		if shouldSkipYAMLConnector(wrapper, isPubliclyAccessible) {
			continue
		}
		cc.startControllerFromConfig(wrapper)
	}
}

func shouldSkipYAMLConnector(wrapper *runtimeconfig.ConnectorConfigWrapper, isPubliclyAccessible bool) bool {
	if !wrapper.Enabled {
		log.Warnf("Connector %s is disabled in configuration", wrapper.ConnectorType)
		return true
	}
	if chatBotProtocols[wrapper.ConnectorType] {
		log.Infof("Skipping YAML connector %s: chat bots are configured via the Web UI", wrapper.ConnectorType)
		return true
	}
	if oauthConnectorTypes[wrapper.ConnectorType] && !isPubliclyAccessible {
		log.Infof("Skipping OAuth connector %s: IsPubliclyAccessible is false", wrapper.ConnectorType)
		return true
	}
	return false
}

func (cc *ConnectionController) startDefaultOAuthConnectors(isPubliclyAccessible bool) {
	if !isPubliclyAccessible {
		log.Infof("OAuth connectors not started: IsPubliclyAccessible is false")
		return
	}
	cc.setupConnector(&mastodon.MastodonConnector{})
	cc.setupConnector(&x.XConnector{})
	cc.setupConnector(&bluesky.BlueskyConnector{})
	cc.setupConnector(&facebook.FacebookConnector{})
	cc.setupConnector(&instagram.InstagramConnector{})
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
	maps.Copy(services, cc.controllers)
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
	startedTypes := cc.collectStartedTypes(available)
	cc.mu.RUnlock()

	isPubliclyAccessible := cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible)
	unregistered := make([]*connector.UnregisteredConnector, 0)

	for _, connectorType := range available {
		if chatBotProtocols[connectorType] || startedTypes[connectorType] {
			continue
		}
		unregistered = append(unregistered, cc.buildUnregisteredConnector(connectorType, isPubliclyAccessible))
	}

	return unregistered
}

func (cc *ConnectionController) collectStartedTypes(available []string) map[string]bool {
	startedTypes := make(map[string]bool)
	for k := range cc.controllers {
		for _, connectorType := range available {
			if controllerKeyMatchesType(k, connectorType) {
				startedTypes[connectorType] = true
				break
			}
		}
	}
	return startedTypes
}

func controllerKeyMatchesType(key, connectorType string) bool {
	return key == connectorType || (len(key) > len(connectorType) && key[:len(connectorType)+1] == connectorType+"-")
}

func (cc *ConnectionController) buildUnregisteredConnector(connectorType string, isPubliclyAccessible bool) *connector.UnregisteredConnector {
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
	conn := cc.createConnectorInstance(connectorType)
	icon := iconMap[connectorType]
	if conn != nil {
		icon = conn.GetIcon()
	}
	return &connector.UnregisteredConnector{
		Protocol:         connectorType,
		Icon:             icon,
		Name:             connectorType,
		NotStartedReason: getNotStartedReason(connectorType, isPubliclyAccessible),
	}
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
	factories := map[string]func() connector.BaseConnector{
		"telegram":  func() connector.BaseConnector { return &telegram.TelegramConnector{} },
		"discord":   func() connector.BaseConnector { return &discord.DiscordConnector{} },
		"mastodon":  func() connector.BaseConnector { return &mastodon.MastodonConnector{} },
		"x":         func() connector.BaseConnector { return &x.XConnector{} },
		"bluesky":   func() connector.BaseConnector { return &bluesky.BlueskyConnector{} },
		"facebook":  func() connector.BaseConnector { return &facebook.FacebookConnector{} },
		"instagram": func() connector.BaseConnector { return &instagram.InstagramConnector{} },
	}
	if factory, ok := factories[connectorType]; ok {
		return factory()
	}
	return nil
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

func (cc *ConnectionController) setupConnector(c connector.BaseConnector) {
	cc.setupConnectorWithKey(c, nil, c.GetProtocol())
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

	if !cc.db.GetCvarBool(db.CvarKeys.IsPubliclyAccessible) {
		cc.stopOAuthConnectors()
		return
	}
	cc.ensureDefaultOAuthConnectors()
}

func (cc *ConnectionController) stopOAuthConnectors() {
	for key := range cc.controllers {
		for oauthType := range oauthConnectorTypes {
			if controllerKeyMatchesType(key, oauthType) {
				delete(cc.controllers, key)
				log.Infof("Stopped OAuth connector: %s (IsPubliclyAccessible is false)", key)
				break
			}
		}
	}
}

func (cc *ConnectionController) ensureDefaultOAuthConnectors() {
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
