package frontend

import (
	"errors"
	"io/fs"
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

// spaHandler serves the SPA by serving index.html for non-API routes that don't exist as files
func spaHandler(webuiDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(webuiDir))
	webui := os.DirFS(webuiDir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !webuiFileExists(webui, r.URL.Path) {
			if isServerRoute(r.URL.Path) {
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

// The lookup goes through an fs.FS, whose paths cannot contain ".." elements,
// so it never looks outside the web UI directory.
func webuiFileExists(webui fs.FS, urlPath string) bool {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		name = "."
	}

	info, err := fs.Stat(webui, name)
	if err == nil && info.IsDir() {
		_, err = fs.Stat(webui, path.Join(name, "index.html"))
	}

	return !errors.Is(err, fs.ErrNotExist)
}

func isServerRoute(urlPath string) bool {
	switch urlPath {
	case "/api", "/oauth2callback", "/lang", "/upload", "/readyz", "/healthz", "/metrics":
		return true
	default:
		return false
	}
}

func GetNewHandler() http.Handler {
	webuiDir := findWebuiDir()
	return spaHandler(webuiDir)
}
