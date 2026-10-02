package llm

// ConnectionPreset is a known OpenAI-compatible provider shortcut for the admin UI.
type ConnectionPreset struct {
	ID             string
	Name           string
	BaseURL        string
	SuggestedModel string
	Description    string
}

const (
	PresetOpenRouter = "openrouter"
	PresetOpenAI     = "openai"
	PresetOllama     = "ollama"
)

// ConnectionPresets are built-in base URL / suggested model shortcuts.
var ConnectionPresets = []ConnectionPreset{
	{
		ID:             PresetOpenRouter,
		Name:           "OpenRouter",
		BaseURL:        "https://openrouter.ai/api/v1",
		SuggestedModel: "openai/gpt-4o-mini",
		Description:    "Unified API for many providers. Use an OpenRouter API key.",
	},
	{
		ID:             PresetOpenAI,
		Name:           "OpenAI",
		BaseURL:        "https://api.openai.com/v1",
		SuggestedModel: "gpt-4o-mini",
		Description:    "Official OpenAI Chat Completions API.",
	},
	{
		ID:             PresetOllama,
		Name:           "Ollama",
		BaseURL:        "http://127.0.0.1:11434/v1",
		SuggestedModel: "llama3.2",
		Description:    "Local Ollama OpenAI-compatible endpoint.",
	},
}

func PresetByID(id string) *ConnectionPreset {
	for i := range ConnectionPresets {
		if ConnectionPresets[i].ID == id {
			return &ConnectionPresets[i]
		}
	}
	return nil
}
