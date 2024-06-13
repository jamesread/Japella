package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jamesread/japella/internal/amqp"

	"gopkg.in/yaml.v2"
)

type CommonConfig struct {
	Amqp *AmqpConfig
}

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

func LoadConfig[V interface{}](filename string, cfg V) {
	log.WithFields(log.Fields{
		"file": filename,
	}).Infof("Loading started")

	err := yaml.UnmarshalStrict(readFile(filename), &cfg)

	log.Infof("Result after UnmarshalStrict %+v", cfg)

	if err != nil {
		log.Fatalf("could not load common config! %v", err)
	}

	log.WithFields(log.Fields {
		"file": filename,
	}).Infof("Loading complete")
}

func LoadConfigCommon(cfg *CommonConfig) {
	cfg.Amqp = &AmqpConfig{}

	LoadConfig("config.common.yaml", cfg)

	log.Infof("LoadConfigCommon AMQP: %+v", cfg.Amqp.Host);

	amqp.AmqpHost = cfg.Amqp.Host
	amqp.AmqpUser = cfg.Amqp.User
	amqp.AmqpPass = cfg.Amqp.Pass
	amqp.AmqpPort = cfg.Amqp.Port
}
