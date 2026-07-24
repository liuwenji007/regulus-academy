package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatStream_accumulatesDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !body.Stream {
			t.Errorf("expected stream=true")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	c := NewOpenAI(OpenAIConfig{Provider: "deepseek", APIKey: "k", BaseURL: srv.URL, Model: "m"})
	var parts []string
	got, err := c.ChatStream(t.Context(), []Message{{Role: "user", Content: "hi"}}, 0.6, func(d string) {
		parts = append(parts, d)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "你好" {
		t.Fatalf("got %q", got)
	}
	if strings.Join(parts, "") != "你好" {
		t.Fatalf("parts=%v", parts)
	}
}
