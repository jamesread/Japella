package i18n

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
)

type LanguageFilev1 struct {
	SchemaVersion int               `json:"schemaVersion"`
	Translations  map[string]string `json:"translations"`
}

type CombinedLanguageContent struct {
	AcceptLanguages []string                     `json:"acceptLanguages"`
	Messages        map[string]map[string]string `json:"messages"`
}

func getLanguageDir() string {
	dirsToSearch := []string{
		"../var/app-skel/lang/",
		"../../../../var/app-skel/lang/", // Relative to this file, for unit tests
		"/app/lang/",
	}

	dir, _ := dirs.GetFirstExistingDirectory("lang", dirsToSearch)

	return dir
}

func createCombineLanguageContent() *CombinedLanguageContent {
	languageDir := getLanguageDir()

	output := &CombinedLanguageContent{
		Messages: make(map[string]map[string]string),
	}

	files, err := os.ReadDir(languageDir)

	if err != nil {
		log.Errorf("Error reading language directory %s: %v", languageDir, err)
		return output
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			languageName := strings.Replace(file.Name(), ".json", "", 1)

			fullPath := filepath.Join(languageDir, file.Name())
			log.Infof("Loading language file: %s", fullPath)

			content, err := os.ReadFile(fullPath)

			if err != nil {
				log.Errorf("Error reading language file %s: %v", fullPath, err)
				continue
			}

			var jsonData LanguageFilev1

			err = json.Unmarshal(content, &jsonData)

			if err != nil {
				log.Errorf("Error reading language file %s: %v", fullPath, err)
				continue
			}

			output.Messages[languageName] = jsonData.Translations
		}
	}

	return output
}

var isLoaded = false
var combinedContent = &CombinedLanguageContent{}

func Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Accept-Language")
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	if !isLoaded || os.Getenv("JAPELLA_DEV_DISABLE_LANGUAGE_CACHE") == "true" {
		combinedContent = createCombineLanguageContent()
		isLoaded = true
	}

	headerLanguage := r.Header.Get("Accept-Language")

	combinedContent.AcceptLanguages = make([]string, 0)

	for _, lang := range strings.Split(headerLanguage, ",") {
		lang = strings.TrimSpace(lang)

		combinedContent.AcceptLanguages = append(combinedContent.AcceptLanguages, lang)
	}

	jsonData, err := json.MarshalIndent(combinedContent, "", "  ")

	if err != nil {
		log.Errorf("Error marshalling combined language content: %v", err)
	}

	w.Write(jsonData)
}
