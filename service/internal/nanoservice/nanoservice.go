package nanoservice

import (
	"os"
	"strings"
)

type Nanoservice interface {
	Start()
}

func GetNanoservices() []string {
	var ret[] string

	services := strings.Split(os.Getenv("JAPELLA_NANOSERVICES"), ",")

	for _, service := range services {
		tmp := strings.TrimSpace(service)

		if tmp != "" {
			ret = append(ret, tmp)
		}
	}

	return ret
}
