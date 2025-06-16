package dashboard

import (
	"net/http"
	//	"net/http/httputil"
	//	"net/url"
	"github.com/jamesread/golure/pkg/dirs"
)

type Dashboard struct {
}

func findWebuiDir() string {
	directoriesToSearch := []string{
		"../frontend/dist/",
		"../webui",
		"/usr/share/Japella/webui/",
	}

	dir, _ := dirs.GetFirstExistingDirectory("webui", directoriesToSearch)

	return dir
}

func GetNewHandler() http.Handler {
	return http.FileServer(http.Dir(findWebuiDir()))
}
