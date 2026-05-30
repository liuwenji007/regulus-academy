package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func setupTestServer(t *testing.T, mockLLM bool) *httptest.Server {
	return setupTestServerWithHandler(t, mockLLM, nil)
}

func setupTestServerWithHandler(t *testing.T, mockLLM bool, llmHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmURL := "https://api.deepseek.com"
	if mockLLM {
		if llmHandler == nil {
			llmHandler = func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"这是测试讲解。回复「开始练习」继续。"}}]}`))
			}
		}
		mock := httptest.NewServer(llmHandler)
		t.Cleanup(mock.Close)
		llmURL = mock.URL
	}

	h, err := NewHandler(store, llm.NewClient("test-key", llmURL))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(NewServer(h, nil, nil))
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

func decodeBuildTree(t *testing.T, resp *http.Response) storage.KnowledgeTree {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if raw, ok := body["tree"]; ok {
		var tree storage.KnowledgeTree
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatal(err)
		}
		return tree
	}
	raw, _ := json.Marshal(body)
	var tree storage.KnowledgeTree
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	return tree
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

	tree := decodeBuildTree(t, resp)
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

func TestBuildDomainGeneratedRust(t *testing.T) {
	chdirToRepo(t)
	treePayload := `{"domain":"Rust","slug":"rust","description":"Rust","layers":{"entry":{"label":"入门","time":"~2h","goal":"g","nodes":[{"key":"rust_basics","title":"基础"},{"key":"ownership","title":"所有权"}]},"intermediate":{"label":"熟悉","time":"~8h","goal":"g","nodes":[{"key":"structs","title":"结构体"},{"key":"enums","title":"枚举"}]},"advanced":{"label":"精通","time":"~20h","goal":"g","nodes":[{"key":"lifetimes","title":"生命周期"},{"key":"async_rust","title":"异步"}]}},"nodes":[{"key":"rust_basics","node":"基础","layer":"入门","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]},{"key":"ownership","node":"所有权","layer":"入门","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]},{"key":"structs","node":"结构体","layer":"熟悉","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]},{"key":"enums","node":"枚举","layer":"熟悉","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]},{"key":"lifetimes","node":"生命周期","layer":"精通","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]},{"key":"async_rust","node":"异步","layer":"精通","core_concepts":["c"],"common_mistakes":["m"],"boundaries":["b"],"exercise_ideas":["e"]}]}`
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(treePayload) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"用户想学 Rust\"}"}}]}`))
	}
	ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"name": "rust"})
	resp, err := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready" {
		t.Fatalf("expected ready, got %+v", result)
	}
	if result["generated"] != true {
		t.Fatalf("expected generated=true, got %+v", result["generated"])
	}
	var tree storage.KnowledgeTree
	if raw, ok := result["tree"]; ok {
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &tree); err != nil {
			t.Fatal(err)
		}
	}
	if len(tree.Layers) != 3 {
		t.Fatalf("layers=%d", len(tree.Layers))
	}
}

func TestSessionFlowWithMockLLM(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	tree := decodeBuildTree(t, resp)
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

func TestListDomainsAPI(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, false)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp1, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	resp1.Body.Close()

	resp, err := http.Get(ts.URL + "/api/domains")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	domains, ok := body["domains"].([]any)
	if !ok || len(domains) < 1 {
		t.Fatalf("expected domains, got %+v", body)
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

func TestBuildDomainIdempotent(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, false)
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp1, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(body))
	tree1 := decodeBuildTree(t, resp1)
	resp1.Body.Close()

	resp2, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(body))
	tree2 := decodeBuildTree(t, resp2)
	resp2.Body.Close()

	if tree1.DomainID != tree2.DomainID {
		t.Fatalf("期望相同 domainId，得到 %s vs %s", tree1.DomainID, tree2.DomainID)
	}
}

func TestActiveSessionResume(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	tree := decodeBuildTree(t, resp)
	resp.Body.Close()

	startBody, _ := json.Marshal(map[string]any{
		"domainId": tree.DomainID,
		"nodeKey":  "goroutine_basics",
		"layer":    "entry",
	})
	resp2, _ := http.Post(ts.URL+"/api/session/start", "application/json", bytes.NewReader(startBody))
	var start1 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&start1)
	resp2.Body.Close()
	sessionID, _ := start1["sessionId"].(string)

	activeURL := ts.URL + "/api/sessions/active?domainId=" + tree.DomainID + "&nodeKey=goroutine_basics"
	resp3, err := http.Get(activeURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	var active map[string]any
	_ = json.NewDecoder(resp3.Body).Decode(&active)
	if active["sessionId"] != sessionID {
		t.Fatalf("active session=%v want %s", active["sessionId"], sessionID)
	}

	resp4, _ := http.Post(ts.URL+"/api/session/start", "application/json", bytes.NewReader(startBody))
	var start2 map[string]any
	_ = json.NewDecoder(resp4.Body).Decode(&start2)
	resp4.Body.Close()
	if start2["resumed"] != true {
		t.Fatal("第二次 start 应返回 resumed")
	}
	if start2["sessionId"] != sessionID {
		t.Fatal("应恢复同一会话")
	}
}

func TestLLMInfo(t *testing.T) {
	ts := setupTestServer(t, false)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/llm/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSessionExerciseJSONFlow(t *testing.T) {
	chdirToRepo(t)
	smartMock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "exercise.json") || strings.Contains(body, "小练习") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"question\":\"1+1=?\",\"question_type\":\"short\",\"reinforced_concepts\":[]}"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"讲解内容。回复开始练习继续。"}}]}`))
	}
	ts := setupTestServerWithHandler(t, true, smartMock)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	tree := decodeBuildTree(t, resp)
	resp.Body.Close()

	startBody, _ := json.Marshal(map[string]any{
		"domainId": tree.DomainID,
		"nodeKey":  "goroutine_basics",
		"layer":    "entry",
	})
	resp2, _ := http.Post(ts.URL+"/api/session/start", "application/json", bytes.NewReader(startBody))
	var startResp map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&startResp)
	resp2.Body.Close()
	sessionID, _ := startResp["sessionId"].(string)

	msgBody, _ := json.Marshal(map[string]string{
		"sessionId": sessionID,
		"content":   "开始练习",
	})
	resp3, err := http.Post(ts.URL+"/api/session/message", "application/json", bytes.NewReader(msgBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("message status=%d", resp3.StatusCode)
	}
	var msgResp map[string]any
	_ = json.NewDecoder(resp3.Body).Decode(&msgResp)
	if msgResp["phase"] != "exercise" {
		t.Fatalf("phase=%v", msgResp["phase"])
	}
}

func TestDeleteDomainAPI(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, false)
	defer ts.Close()

	buildBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	resp, _ := http.Post(ts.URL+"/api/domain/build", "application/json", bytes.NewReader(buildBody))
	tree := decodeBuildTree(t, resp)
	resp.Body.Close()

	wrongBody, _ := json.Marshal(map[string]string{"confirmName": "错误名称"})
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/domain/"+tree.DomainID, bytes.NewReader(wrongBody))
	req.Header.Set("Content-Type", "application/json")
	respBad, _ := http.DefaultClient.Do(req)
	respBad.Body.Close()
	if respBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("错误确认名 status=%d", respBad.StatusCode)
	}

	delBody, _ := json.Marshal(map[string]string{"confirmName": tree.DomainName})
	req2, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/domain/"+tree.DomainID, bytes.NewReader(delBody))
	req2.Header.Set("Content-Type", "application/json")
	respDel, _ := http.DefaultClient.Do(req2)
	respDel.Body.Close()
	if respDel.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d", respDel.StatusCode)
	}

	respList, _ := http.Get(ts.URL + "/api/domains")
	defer respList.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(respList.Body).Decode(&body)
	domains, _ := body["domains"].([]any)
	for _, d := range domains {
		m, _ := d.(map[string]any)
		if m["id"] == tree.DomainID {
			t.Fatal("课程应已从列表移除")
		}
	}
}

func TestGatewayInfoAPI(t *testing.T) {
	ts := setupTestServer(t, false)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/gateway/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["settings"]; !ok {
		t.Fatal("expected settings in response")
	}
	platforms, ok := body["platforms"].([]any)
	if !ok || len(platforms) != 4 {
		t.Fatalf("expected 4 platforms, got %+v", body["platforms"])
	}
}

func readBody(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}
