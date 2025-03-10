package dashboard

import (
	"net/http"
	//	"net/http/httputil"
	//	"net/url"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type Dashboard struct {
}

func findWebuiDir() string {
	directoriesToSearch := []string{
		"../webui",
		"/usr/share/Japella/webui/",
	}

	for i := 0; i < len(directoriesToSearch); i++ {
		dir := directoriesToSearch[i]
		absdir, _ := filepath.Abs(dir)

		if _, err := os.Stat(absdir); !os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"dir": absdir,
			}).Infof("Found the webui directory")

			return dir
		}
	}

	log.Warnf("Did not find the webui directory, you will probably get 404 errors.")

	return "./webui" // should not exist
}

func GetNewHandler() http.Handler {
	return http.FileServer(http.Dir(findWebuiDir()))
}
