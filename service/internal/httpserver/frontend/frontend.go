package frontend

import (
	"net/http"
	"os"
	"path/filepath"

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
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists
		path := filepath.Join(webuiDir, r.URL.Path)
		
		// If the path is a directory, check for index.html
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			path = filepath.Join(path, "index.html")
		}
		
		// If the file doesn't exist, serve index.html for SPA routing
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Check if this is an API route or other non-SPA route
			if r.URL.Path == "/api" || 
			   r.URL.Path == "/oauth2callback" || 
			   r.URL.Path == "/lang" || 
			   r.URL.Path == "/upload" || 
			   r.URL.Path == "/readyz" || 
			   r.URL.Path == "/healthz" || 
			   r.URL.Path == "/metrics" {
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

func GetNewHandler() http.Handler {
	webuiDir := findWebuiDir()
	return spaHandler(webuiDir)
}
