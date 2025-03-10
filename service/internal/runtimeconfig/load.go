package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"sync"
)

var cfg *CommonConfig

func findFile(filename string) string {
	paths := []string{
		"../",
		"/config/",
	}

	for _, path := range paths {
		absPath, err := filepath.Abs(filepath.Join(path, filename))

		if err != nil {
			log.Warnf("Failed to get the absolute path for %v / %v", path, filename)
		}

		if _, err := os.Stat(absPath); err == nil {
			log.Infof("Config found %v at %v", filename, absPath)

			return absPath
		} else {
			log.Infof("Didn't find %v at %v", filename, absPath)
		}
	}

	return filename
}

func readFile(filename string) []byte {
	filename = findFile(filename)

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

func getConfigFilename() string {
	configFilename := os.Getenv("CONFIG_FILE")

	if configFilename == "" {
		configFilename = "config.yaml"
	}

	return configFilename
}

func Get() *CommonConfig {
	cfgGetLock.Lock()

	if cfg == nil {
		cfg = &CommonConfig{}
		cfg.Amqp = &AmqpConfig{}
		cfg.Connectors = &ConnectorConfig{}
		cfg.Connectors.Discord = &DiscordConfig{}
		cfg.Connectors.Telegram = &TelegramConfig{}

		loadConfig(getConfigFilename())
	}

	cfgGetLock.Unlock()

	return cfg
}

func loadConfig(filename string) *CommonConfig {
	log.WithFields(log.Fields{
		"file": filename,
	}).Infof("Config loading started")

	err := yaml.UnmarshalStrict(readFile(filename), &cfg)

	if err != nil {
		log.Fatalf("could not load common config! %v", err)
	}

	log.WithFields(log.Fields{
		"file": filename,
	}).Infof("Config loading complete")

	return cfg
}
