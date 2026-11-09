package frontend

import (
	"net/http"

	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/golure/pkg/spa"
)

func findWebuiDir() string {
	directoriesToSearch := []string{
		"../frontend/dist/",
		"../../webui",
		"/usr/share/Japella/webui/",
		"../../../frontend/dist/", // Relative to this file, for unit tests
	}

	dir, _ := dirs.GetFirstExistingDirectory("webui", directoriesToSearch)

	return dir
}

func GetNewHandler() http.Handler {
	// Routes owned by the main server must 404 here rather than load the SPA.
	return spa.Handler(findWebuiDir(), spa.Options{
		Exclude: []string{"/api", "/oauth2callback", "/lang", "/upload", "/readyz", "/healthz", "/metrics"},
	})
}
