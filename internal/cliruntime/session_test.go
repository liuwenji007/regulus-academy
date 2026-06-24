package cliruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStartLinkedWithoutLocalLLM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/domains":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"domains": []map[string]string{
					{"id": "dom-1", "slug": "go-concurrency", "name": "Go 并发"},
				},
			})
		case "/api/session/start":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "sess-remote",
				"domainId":  "dom-1",
				"nodeKey":   "goroutine_basics",
				"phase":     "explain",
				"content":   "远程讲解",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	coach := filepath.Join(t.TempDir(), "coach")
	if err := os.MkdirAll(filepath.Join(coach, ".regulus"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := []byte(`{"apiUrl":"` + srv.URL + `","userId":"default"}`)
	if err := os.WriteFile(filepath.Join(coach, ".regulus", "link.json"), link, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LLM_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	rt, err := Open(coach)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	if rt.LLMConfigured() {
		t.Fatal("测试需要未配置本地 LLM")
	}

	out, err := rt.SessionStart(context.Background(), "go-concurrency", "goroutine_basics", "entry")
	if err != nil {
		t.Fatalf("SessionStart linked: %v", err)
	}
	if out.SessionID != "sess-remote" || out.Content != "远程讲解" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestSessionMessageLinkedWithoutLocalLLM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/session/message":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "sess-remote",
				"phase":     "exercise",
				"content":   "题目",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	coach := filepath.Join(t.TempDir(), "coach")
	if err := os.MkdirAll(filepath.Join(coach, ".regulus"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := []byte(`{"apiUrl":"` + srv.URL + `","userId":"default"}`)
	if err := os.WriteFile(filepath.Join(coach, ".regulus", "link.json"), link, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLM_API_KEY", "")

	rt, err := Open(coach)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	out, err := rt.SessionMessage(context.Background(), "sess-remote", "开始练习")
	if err != nil {
		t.Fatalf("SessionMessage linked: %v", err)
	}
	if out.Phase != "exercise" {
		t.Fatalf("phase=%q", out.Phase)
	}
}
