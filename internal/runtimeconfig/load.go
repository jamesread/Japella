package runtimeconfig

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"

	"github.com/jamesread/japella/internal/amqp"

	"gopkg.in/yaml.v2"
)

type CommonConfig struct {
	Amqp AmqpConfig
}

func readFile(filename string) []byte {
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

func LoadConfig(filename string, cfg interface{}) {
	err := yaml.UnmarshalStrict(readFile(filename), cfg)

	log.Infof("loaded %+v", cfg)
	if err != nil {
		log.Fatalf("could not load common config!")
	}

	log.Infof("Loaded config: %v", filename)
}

func LoadConfigCommon(cfg *CommonConfig) {
	LoadConfig("config.common.yaml", cfg)

	//	log.Infof("after %+v", cfg)

	amqp.AmqpHost = cfg.Amqp.Host
	amqp.AmqpUser = cfg.Amqp.User
	amqp.AmqpPass = cfg.Amqp.Pass
	amqp.AmqpPort = cfg.Amqp.Port
}
