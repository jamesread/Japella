package frontend

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/jamesread/golure/pkg/dirs"
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

var nonSPAExactPaths = map[string]bool{
	"/api":            true,
	"/oauth2callback": true,
	"/lang":           true,
	"/upload":         true,
	"/readyz":         true,
	"/healthz":        true,
	"/metrics":        true,
}

// spaHandler serves the SPA by serving index.html for non-API routes that don't exist as files
func spaHandler(webuiDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(webuiDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !webuiFileExists(webuiDir, r.URL.Path) {
			if nonSPAExactPaths[r.URL.Path] {
				// Let the main server handle these routes
				http.NotFound(w, r)
				return
			}
			// For all other routes, serve index.html to enable client-side routing
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// webuiFileExists reports whether urlPath names a file in webuiDir, or a
// directory with an index.html. os.Root keeps the lookup inside webuiDir.
func webuiFileExists(webuiDir, urlPath string) bool {
	root, err := os.OpenRoot(webuiDir)
	if err != nil {
		return false
	}
	defer func() { _ = root.Close() }()

	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		name = "."
	}
	info, err := root.Stat(name)
	if err == nil && info.IsDir() {
		_, err = root.Stat(path.Join(name, "index.html"))
	}
	return err == nil
}

func GetNewHandler() http.Handler {
	webuiDir := findWebuiDir()
	return spaHandler(webuiDir)
}
