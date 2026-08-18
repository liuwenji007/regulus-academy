package llm

import (
	"encoding/json"
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

func TestExtractJSON_unclosedFence(t *testing.T) {
	raw := "```json\n{\"domain\":\"心理学\",\"slug\":\"psychology\"}\n"
	got := extractJSON(raw)
	if got != `{"domain":"心理学","slug":"psychology"}` {
		t.Fatalf("unclosed fence should still extract object, got %q", got)
	}
	var aux map[string]string
	if err := json.Unmarshal([]byte(got), &aux); err != nil {
		t.Fatal(err)
	}
}

func TestExtractJSON_plainObject(t *testing.T) {
	got := extractJSON(`{"ok":true}`)
	if got != `{"ok":true}` {
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

func TestChatRetriesOnEmptyContent(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"开场讲解"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{Provider: "deepseek", APIKey: "k", BaseURL: srv.URL, Model: "m"})
	got, err := c.Chat(t.Context(), []Message{{Role: "user", Content: "begin"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "开场讲解" || calls != 2 {
		t.Fatalf("expected retry success, got %q calls=%d", got, calls)
	}
}

func TestChatErrorsWhenAlwaysEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"  "}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{Provider: "deepseek", APIKey: "k", BaseURL: srv.URL, Model: "m"})
	_, err := c.Chat(t.Context(), []Message{{Role: "user", Content: "begin"}})
	if err == nil || !strings.Contains(err.Error(), "空内容") {
		t.Fatalf("expected empty-content error, got %v", err)
	}
}

func TestChatPromptJSONSkipsResponseFormat(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"reply\":\"你好\",\"ready_to_plan\":false}"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{
		Provider: "deepseek",
		APIKey:   "test",
		BaseURL:  srv.URL,
		Model:    "deepseek-chat",
	})
	var out struct {
		Reply       string `json:"reply"`
		ReadyToPlan bool   `json:"ready_to_plan"`
	}
	err := ChatPromptJSON(t.Context(), c, []Message{{Role: "user", Content: "hi"}}, 0.3, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reply != "你好" {
		t.Fatalf("reply=%q", out.Reply)
	}
	if strings.Contains(captured, `"response_format"`) {
		t.Fatalf("ChatPromptJSON should not use json_object mode: %s", captured)
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

func TestChatJSONRetryRequestErrorSurfacesRetryFailure(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json at all"}}]}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream down"}`))
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
	if err == nil {
		t.Fatal("expected error")
	}
	if calls < 2 {
		t.Fatalf("expected retry request, calls=%d", calls)
	}
	if !strings.Contains(err.Error(), "502") && !strings.Contains(err.Error(), "Bad Gateway") {
		t.Fatalf("应暴露重试请求失败而非首次解析错误，got %v", err)
	}
	if strings.Contains(err.Error(), "not json at all") {
		t.Fatalf("不应把首次 unmarshal 内容当作最终错误: %v", err)
	}
}

func TestChatJSONAcceptsControlCharInString(t *testing.T) {
	inner := "{\"points\":[\"hello\x18world\"]}"
	payload, err := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": inner}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{Provider: "custom", APIKey: "k", BaseURL: srv.URL, Model: "hy3-preview"})
	var out struct {
		Points []string `json:"points"`
	}
	if err := c.ChatJSON(t.Context(), []Message{{Role: "user", Content: "map"}}, 0.1, &out); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("control char should parse without retry, calls=%d", calls)
	}
	if len(out.Points) != 1 || out.Points[0] != "hello\x18world" {
		t.Fatalf("points=%q", out.Points)
	}
}
