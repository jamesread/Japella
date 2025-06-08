package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"sync"

	"errors"
	"fmt"
	"github.com/jamesread/golure/pkg/dirs"
)

var cfg *CommonConfig

func readFile(filename string) []byte {
	filename = filepath.Join(getConfigPath(), filename)

	handle, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Load %v", err)
	}

	content, err := io.ReadAll(handle)

	if err != nil {
		log.Fatalf("Load %v", err)
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

	log.Infof("Connector type: %v", typeHolder.Type)

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
	case "mastodon":
		var v MastodonConfig

		if err := yaml.NodeToValue(typeHolder.Config, &v, yaml.Strict()); err != nil {
			return err
		}

		w.ConnectorConfig = &v
	default:
		return errors.New(fmt.Sprintf("unknown connector type :%v", typeHolder.Type))
	}

	return nil
}

func loadConfig() *CommonConfig {
	log.Infof("Config loading started")

	cfg = &CommonConfig{}

	err := yaml.UnmarshalWithOptions(readFile("config.yaml"), cfg, yaml.Strict())

	if err != nil {
		log.Fatalf("could not load common config! %v", err)
	}

	log.Infof("Config loading complete")

	return cfg
}
