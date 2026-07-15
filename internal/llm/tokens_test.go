package llm

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJsonRetryUserMessage_invalidCharacterBranches(t *testing.T) {
	quoteMsg := jsonRetryUserMessage(errors.New("invalid character 'h' after object key:value pair"))
	if !strings.Contains(quoteMsg, "转义") && !strings.Contains(quoteMsg, `\"`) {
		t.Fatalf("key:value retry should mention escaping: %q", quoteMsg)
	}
	if strings.Contains(quoteMsg, "markdown 代码块") {
		t.Fatalf("key:value retry should not blame markdown: %q", quoteMsg)
	}

	arrayMsg := jsonRetryUserMessage(errors.New("invalid character 'a' after array element"))
	if !strings.Contains(arrayMsg, "逗号") {
		t.Fatalf("array retry should mention comma: %q", arrayMsg)
	}
	if strings.Contains(arrayMsg, "markdown 代码块") {
		t.Fatalf("array retry should not blame markdown: %q", arrayMsg)
	}

	otherMsg := jsonRetryUserMessage(errors.New("invalid character 'x' looking for beginning of value"))
	if strings.Contains(otherMsg, "markdown 代码块") {
		t.Fatalf("generic invalid-character retry should not blame markdown: %q", otherMsg)
	}
	if !strings.Contains(otherMsg, "合法 JSON") {
		t.Fatalf("generic invalid-character retry: %q", otherMsg)
	}
}

func TestMaxTokensFromEnv_default(t *testing.T) {
	t.Setenv("REGULUS_LLM_MAX_TOKENS", "")
	if got := MaxTokensFromEnv(); got != defaultJSONMaxTokens {
		t.Fatalf("default=%d got=%d", defaultJSONMaxTokens, got)
	}
}

func TestMaxTokensFromEnv_zeroMeansUnlimited(t *testing.T) {
	t.Setenv("REGULUS_LLM_MAX_TOKENS", "0")
	if got := MaxTokensFromEnv(); got != 0 {
		t.Fatalf("0 should mean unlimited, got=%d", got)
	}
}

func TestDomainBuildMaxTokensFromEnv_defaultUnlimited(t *testing.T) {
	t.Setenv("REGULUS_LLM_MAX_TOKENS", "")
	t.Setenv("REGULUS_DOMAIN_BUILD_MAX_TOKENS", "")
	if got := DomainBuildMaxTokensFromEnv(); got != 0 {
		t.Fatalf("build default should be unlimited (0), got=%d", got)
	}
}

func TestDomainBuildMaxTokensFromEnv_inheritsLLMWhenSet(t *testing.T) {
	t.Setenv("REGULUS_LLM_MAX_TOKENS", "16384")
	t.Setenv("REGULUS_DOMAIN_BUILD_MAX_TOKENS", "")
	if got := DomainBuildMaxTokensFromEnv(); got != 16384 {
		t.Fatalf("should inherit LLM max, got=%d", got)
	}
}

func TestChatJSONRequestUsesHigherMaxTokens(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"passed\":true}"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("REGULUS_LLM_MAX_TOKENS", "8192")
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
	if !strings.Contains(captured, `"max_tokens":8192`) {
		t.Fatalf("expected max_tokens=8192 in request: %s", captured)
	}
}

func TestChatJSONDomainBuildOmitsMaxTokensWhenUnlimited(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"passed\":true}"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{
		Provider:  "deepseek",
		APIKey:    "test",
		BaseURL:   srv.URL,
		Model:     "deepseek-chat",
		MaxTokens: 8192,
	})
	ctx := WithJSONMaxTokens(t.Context(), 0)
	var out struct {
		Passed bool `json:"passed"`
	}
	if err := c.ChatJSON(ctx, []Message{{Role: "user", Content: "tree"}}, 0.2, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(captured, `"max_tokens"`) {
		t.Fatalf("unlimited build should omit max_tokens: %s", captured)
	}
}

func TestChatJSONTruncationRetryBumpsMaxTokens(t *testing.T) {
	calls := 0
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"passed\":true"` + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"passed\":true}"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("REGULUS_LLM_MAX_TOKENS", "8192")
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
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if !strings.Contains(bodies[1], `"max_tokens":16384`) {
		t.Fatalf("retry should bump max_tokens: %s", bodies[1])
	}
	if !strings.Contains(bodies[1], "截断") {
		t.Fatalf("retry should mention truncation: %s", bodies[1])
	}
}

func TestWithJSONMaxTokensOverridesDefault(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{
		Provider:  "deepseek",
		APIKey:    "test",
		BaseURL:   srv.URL,
		Model:     "deepseek-chat",
		MaxTokens: 4096,
	})
	ctx := WithJSONMaxTokens(t.Context(), 12288)
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.ChatJSON(ctx, []Message{{Role: "user", Content: "x"}}, 0.2, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(captured, `"max_tokens":12288`) {
		t.Fatalf("context override missing: %s", captured)
	}
}
