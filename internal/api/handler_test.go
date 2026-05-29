package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func setupTestServer(t *testing.T, mockLLM bool) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmURL := "https://api.deepseek.com"
	if mockLLM {
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"这是测试讲解。回复「开始练习」继续。"}}]}`))
		}))
		t.Cleanup(mock.Close)
		llmURL = mock.URL
	}

	h, err := NewHandler(store, llm.NewClient("test-key", llmURL))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(NewServer(h, nil))
}

func chdirToRepo(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			if err := os.Chdir(d); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(wd) })
			return
		}
	}
	t.Fatal("找不到 regulus-coach 目录")
}

func TestHealth(t *testing.T) {
	ts := setupTestServer(t, false)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestBuildDomainAndTree(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, false)
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, err := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("build status=%d", resp.StatusCode)
	}

	var tree storage.KnowledgeTree
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		t.Fatal(err)
	}
	if tree.DomainID == "" {
		t.Fatal("缺少 domainId")
	}
	if len(tree.Layers) != 3 {
		t.Fatalf("期望 3 层，得到 %d", len(tree.Layers))
	}

	resp2, err := http.Get(ts.URL + "/api/domain/" + tree.DomainID + "/tree")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("tree status=%d", resp2.StatusCode)
	}
}

func TestSessionFlowWithMockLLM(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	var tree storage.KnowledgeTree
	_ = json.NewDecoder(resp.Body).Decode(&tree)
	resp.Body.Close()

	startBody, _ := json.Marshal(map[string]any{
		"domainId": tree.DomainID,
		"nodeKey":  "goroutine_basics",
		"layer":    "entry",
	})
	resp2, err := http.Post(ts.URL+"/api/session/start", "application/json", bytes.NewReader(startBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("start status=%d", resp2.StatusCode)
	}
	var startResp map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&startResp)
	if startResp["content"] == "" {
		t.Fatal("期望开场讲解")
	}

	sessionID, _ := startResp["sessionId"].(string)
	msgBody, _ := json.Marshal(map[string]string{
		"sessionId": sessionID,
		"content":   "什么是 goroutine？",
	})
	resp3, err := http.Post(ts.URL+"/api/session/message", "application/json", bytes.NewReader(msgBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("message status=%d", resp3.StatusCode)
	}
}

func TestUserProgress(t *testing.T) {
	ts := setupTestServer(t, false)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/user/progress")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
