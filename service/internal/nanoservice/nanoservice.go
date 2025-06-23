package nanoservice

import (
	"strings"

	"github.com/jamesread/japella/internal/runtimeconfig"
)

type Nanoservice interface {
	Start()
}

func GetNanoservices() []string {
	var ret []string

	services := runtimeconfig.Get().Nanoservices

	for _, service := range services {
		tmp := strings.TrimSpace(service.Name)

		if tmp != "" {
			ret = append(ret, tmp)
		}
	}

	return ret
}
