package runtimeconfig

import (
	"os"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
)

func Load(filename string) []byte {
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
