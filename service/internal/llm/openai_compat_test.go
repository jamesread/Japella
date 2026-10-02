package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	got, err := NormalizeBaseURL(" https://api.openai.com/ ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://api.openai.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeBaseURL("ftp://example.com"); err == nil {
		t.Fatal("expected error for ftp")
	}
	if _, err := NormalizeBaseURL(""); err == nil {
		t.Fatal("expected error for empty")
	}
}

func TestCompletionsURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"http://localhost:11434/v1/chat/completions", "http://localhost:11434/v1/chat/completions"},
	}
	for _, tc := range cases {
		got, err := CompletionsURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestChatCompletionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		var req chatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-test" {
			t.Fatalf("model %q", req.Model)
		}
		if len(req.Messages) != 2 || req.Messages[0].Role != "system" || req.Messages[1].Role != "user" {
			t.Fatalf("messages %+v", req.Messages)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": " positive "}},
			},
		})
	}))
	defer srv.Close()

	client := NewClient()
	client.HTTPClient = srv.Client()
	out, err := client.ChatCompletion(context.Background(), srv.URL, "test-key", "gpt-test", []ChatMessage{
		{Role: "system", Content: "Analyze sentiment"},
		{Role: "user", Content: "Hello world"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "positive" {
		t.Fatalf("got %q", out)
	}
}

func TestChatCompletionAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "invalid api key"},
		})
	}))
	defer srv.Close()

	client := NewClient()
	client.HTTPClient = srv.Client()
	_, err := client.ChatCompletion(context.Background(), srv.URL, "bad", "gpt-test", []ChatMessage{
		{Role: "user", Content: "hi"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("got %v", err)
	}
}

func TestChatCompletionHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	client := NewClient()
	client.HTTPClient = srv.Client()
	_, err := client.ChatCompletion(context.Background(), srv.URL, "", "gpt-test", []ChatMessage{
		{Role: "user", Content: "hi"},
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("got %v", err)
	}
}

func TestModelsURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://openrouter.ai/api/v1", "https://openrouter.ai/api/v1/models"},
		{"https://api.openai.com", "https://api.openai.com/v1/models"},
		{"http://127.0.0.1:11434/v1/models", "http://127.0.0.1:11434/v1/models"},
	}
	for _, tc := range cases {
		got, err := ModelsURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestListModelsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "openai/gpt-4o-mini", "name": "GPT-4o Mini"},
				{"id": "anthropic/claude-3.5-sonnet"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient()
	client.HTTPClient = srv.Client()
	models, err := client.ListModels(context.Background(), srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models", len(models))
	}
	if models[0].ID != "openai/gpt-4o-mini" || models[0].Name != "GPT-4o Mini" {
		t.Fatalf("first model %+v", models[0])
	}
	if models[1].Name != "anthropic/claude-3.5-sonnet" {
		t.Fatalf("second name %q", models[1].Name)
	}
}

func TestPresetByID(t *testing.T) {
	p := PresetByID(PresetOpenRouter)
	if p == nil || p.BaseURL == "" {
		t.Fatal("expected openrouter preset")
	}
	if PresetByID("missing") != nil {
		t.Fatal("expected nil for unknown preset")
	}
}
