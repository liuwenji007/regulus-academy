package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearch_parsesBrave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			t.Fatalf("token=%s", r.Header.Get("X-Subscription-Token"))
		}
		if !strings.Contains(r.URL.Query().Get("q"), "计费") {
			t.Fatalf("q=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"文档","url":"https://example.com/a","description":"阈值 42"}]}}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "test-key", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := c.Search(context.Background(), "计费阈值")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Snippet != "阈值 42" {
		t.Fatalf("%+v", got)
	}
	prompt := FormatWebPrompt("计费阈值", got)
	if !strings.Contains(prompt, "[W1]") || !strings.Contains(prompt, "查询词") {
		t.Fatalf("prompt=%s", prompt)
	}
}

func TestConfigured_falseWithoutKey(t *testing.T) {
	t.Setenv("SEARCH_API_KEY", "")
	if Configured() {
		t.Fatal("expected unconfigured")
	}
}
