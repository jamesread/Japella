package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"sync"

	"github.com/jamesread/golure/pkg/dirs"
)

var cfg *CommonConfig

func getConfigFilePath(filename string) string {
	filename = filepath.Join(getConfigPath(), filename)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Warnf("Config file %s does not exist", filename)
	}

	return filename
}

func readFile(filename string) []byte {
	handle, err := os.Open(filename)

	if err != nil {
		log.Warnf("Load %v", err)
	}

	content, err := io.ReadAll(handle)

	if err != nil {
		log.Warnf("Load %v", err)
	}

	return content
}

var cfgGetLock sync.RWMutex

func getConfigPath() string {
	paths := []string{
		"/config/",
		"~/.config/japella/",
		"../",
	}

	selected, _ := dirs.GetFirstExistingDirectory("config", paths)

	return selected
}

func Get() *CommonConfig {
	cfgGetLock.Lock()

	if cfg == nil {
		cfg = &CommonConfig{}

		loadConfig()
	}

	cfgGetLock.Unlock()

	return cfg
}

//gocyclo:ignore
func (w *ConnectorConfigWrapper) UnmarshalYAML(node ast.Node) error {
	var typeHolder struct {
		Type    string
		Enabled bool
		Config  ast.Node
	}

	if err := yaml.NodeToValue(node, &typeHolder); err != nil {
		log.Errorf("could not unmarshal connector type: %v", err)
		return err
	}

	log.Infof("Handling connector type from config: %v", typeHolder.Type)

	w.ConnectorType = typeHolder.Type
	w.Enabled = typeHolder.Enabled

	switch typeHolder.Type {
	case "discord":
		var v DiscordConfig

		if err := yaml.NodeToValue(typeHolder.Config, &v, yaml.Strict()); err != nil {
			return err
		}

		w.ConnectorConfig = &v
	case "telegram":
		var v TelegramConfig

		if err := yaml.NodeToValue(typeHolder.Config, &v, yaml.Strict()); err != nil {
			return err
		}

		w.ConnectorConfig = &v
	case "whatsapp":
		var v WhatsAppConfig

		if err := yaml.NodeToValue(typeHolder.Config, &v, yaml.Strict()); err != nil {
			return err
		}

		w.ConnectorConfig = &v
	case "bluesky":
		var v BlueskyConfig

		if err := yaml.NodeToValue(typeHolder.Config, &v, yaml.Strict()); err != nil {
			return err
		}

		w.ConnectorConfig = &v
	default:
		log.Warnf("Connector type is unknown: %v", typeHolder.Type)

		return nil
	}

	return nil
}

func loadEnvVars(cfg *CommonConfig) {
	loadEnvVarStr(&cfg.Database.Host, "JAPELLA_DB_HOST")
	loadEnvVarStr(&cfg.Database.User, "JAPELLA_DB_USER")
	loadEnvVarStr(&cfg.Database.Pass, "JAPELLA_DB_PASS")
	loadEnvVarInt(&cfg.Database.Port, "JAPELLA_DB_PORT")
	loadEnvVarStr(&cfg.Database.Name, "JAPELLA_DB_NAME")
}

func loadEnvVarInt(variable *int, envVar string) {
	value := os.Getenv(envVar)

	if value != "" {
		log.Infof("Overriding config variable with environment variable: %s", envVar)
		intValue, err := strconv.Atoi(value)
		if err != nil {
			log.Errorf("Invalid integer value for %s: %v", envVar, err)
			return
		}
		*variable = intValue
	}
}

func loadEnvVarStr(variable *string, envVar string) {
	value := os.Getenv(envVar)

	if value != "" {
		log.Infof("Overriding config variable with environment variable: %s", envVar)
		*variable = value
	}
}

func loadConfig() *CommonConfig {
	configFilename := getConfigFilePath("config.yaml")

	log.WithFields(log.Fields{
		"filename": configFilename,
	}).Infof("Loading config file")

	cfg = &CommonConfig{}

	err := yaml.UnmarshalWithOptions(readFile(configFilename), cfg, yaml.Strict())

	if err != nil {
		log.Warnf("could not load common config! %v", err)
	}

	loadEnvVars(cfg)

	log.Infof("Config loading complete")

	return cfg
}
