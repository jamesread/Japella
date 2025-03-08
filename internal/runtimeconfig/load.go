package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"sync"
)

var cfg *CommonConfig

func findFile(filename string) string {
	paths := []string {
		"./",
		"/config/",
	}

	for _, path := range paths {
		absPath, _ := filepath.Abs(filepath.Join(path, filename))

		if _, err := os.Stat(absPath); err == nil {
			log.Infof("Found %v at %v", filename, absPath)

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

	content, err := ioutil.ReadAll(handle)

	if err != nil {
		log.Fatalf("Load %v", err)
	}

	return content
}

var cfgGetLock sync.RWMutex

func Get() *CommonConfig {
	cfgGetLock.Lock()

	if cfg == nil {
		cfg = &CommonConfig{}
		cfg.Amqp = &AmqpConfig{}
		cfg.Connectors = &ConnectorConfig{}
		cfg.Connectors.Discord = &DiscordConfig{}
		cfg.Connectors.Telegram = &TelegramConfig{}

		loadConfig("config.yaml")
	}

	cfgGetLock.Unlock()

	return cfg;
}

func loadConfig(filename string) *CommonConfig {
	log.WithFields(log.Fields{
		"file": filename,
	}).Infof("Loading started")

	err := yaml.UnmarshalStrict(readFile(filename), &cfg)

	if err != nil {
		log.Fatalf("could not load common config! %v", err)
	}

	log.WithFields(log.Fields {
		"file": filename,
	}).Infof("Loading complete")
	
	return cfg
}
