package llm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	raw := "```json\n{\"a\":1}\n```"
	got := extractJSON(raw)
	if got != `{"a":1}` {
		t.Fatalf("got %q", got)
	}
}

func TestSupportsJSONMode(t *testing.T) {
	c := NewOpenAI(OpenAIConfig{Provider: "deepseek", APIKey: "k", BaseURL: "https://api.deepseek.com", Model: "m"})
	if !c.supportsJSONMode() {
		t.Fatal("deepseek should support json mode")
	}
	o := NewOpenAI(OpenAIConfig{Provider: "ollama", BaseURL: "http://localhost:11434", Model: "m"})
	if o.supportsJSONMode() {
		t.Fatal("ollama should not use json mode")
	}
}

func TestChatJSONRequestIncludesResponseFormat(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"passed\":true,\"feedback\":\"ok\",\"mistake_concepts\":[]}"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{
		Provider: "deepseek",
		APIKey:   "test",
		BaseURL:  srv.URL,
		Model:    "deepseek-chat",
	})
	var out struct {
		Passed bool `json:"passed"`
	}
	err := c.ChatJSON(t.Context(), []Message{{Role: "user", Content: "grade"}}, 0.2, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(captured, `"response_format"`) {
		t.Fatalf("expected response_format in request body: %s", captured)
	}
	if !strings.Contains(captured, `"json_object"`) {
		t.Fatalf("expected json_object in request body: %s", captured)
	}
}
