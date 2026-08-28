package themes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/jamesread/golure/pkg/dirs"
)

const (
	themesIndexFile             = "themes/index.json"
	supplementalThemesIndexFile = "supplemental-themes/index.json"
)

type indexFile struct {
	Themes []json.RawMessage `json:"themes"`
}

func webuiSearchPaths() []string {
	return []string{
		"../frontend/dist/",
		"../../webui",
		"/usr/share/Japella/webui/",
		"../../../frontend/dist/",
	}
}

func findWebuiDir() string {
	dir, _ := dirs.GetFirstExistingDirectory("webui", webuiSearchPaths())
	return dir
}

func parseThemeID(raw json.RawMessage) string {
	var id string
	if err := json.Unmarshal(raw, &id); err == nil && id != "" {
		return id
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.ID
	}
	return ""
}

func readThemeNamesFromIndex(webuiDir, relativePath string, seen map[string]struct{}, names *[]string) {
	path := filepath.Join(webuiDir, relativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return
	}

	for _, entry := range idx.Themes {
		id := parseThemeID(entry)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		*names = append(*names, id)
	}
}

// AvailableThemeNames returns drop-in theme ids from webui/themes and bundled supplemental themes.
func AvailableThemeNames() []string {
	webuiDir := findWebuiDir()
	if webuiDir == "" {
		return nil
	}

	seen := map[string]struct{}{}
	var names []string
	readThemeNamesFromIndex(webuiDir, themesIndexFile, seen, &names)
	readThemeNamesFromIndex(webuiDir, supplementalThemesIndexFile, seen, &names)
	sort.Strings(names)
	return names
}

func IsValidThemeName(name string, catalog []string) bool {
	if name == "" {
		return true
	}
	for _, allowed := range catalog {
		if allowed == name {
			return true
		}
	}
	return false
}
