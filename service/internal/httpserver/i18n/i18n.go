package i18n

import (
	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"encoding/json"
)

func getLanguageDir() string {
	dirsToSearch := []string{
		"../lang/",
		"/app/lang/",
	}

	dir, _ := dirs.GetFirstExistingDirectory("lang", dirsToSearch)

	return dir
}

func getLanguageFile(name string) (string, error) {
	languageDir := getLanguageDir()

	if languageDir == "" {
		return "404.json", fmt.Errorf("no language directory found")
	}

	log.Infof("getLanguageFile: %v", name);

	languageFilename, err := filepath.Abs(filepath.Join(languageDir, name + ".json"))

	if err != nil {
		return "", fmt.Errorf("error getting absolute path for language file: %v", err)
	}

	if _, err := os.Stat(languageFilename); os.IsNotExist(err) {
		return "", fmt.Errorf("language file does not exist: %s", languageFilename)
	}

	return languageFilename, nil
}

type LanguageFilev1 struct {
	SchemaVersion int			   `json:"schemaVersion"`
	Translations map[string]string `json:"translations"`
}

type CombinedLanguageContent struct {
	AcceptLanguages []string `json:"acceptLanguages"`
	Messages map[string]map[string]string `json:"messages"`
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

	if !isLoaded || os.Getenv("LANGUAGE_CACHE") == "false" {
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
	return
}
