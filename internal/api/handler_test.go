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
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func setupTestServer(t *testing.T, mockLLM bool) *httptest.Server {
	_, _, srv := setupTestServerStore(t, mockLLM, nil)
	return srv
}

func setupTestServerStore(t *testing.T, mockLLM bool, llmHandler http.HandlerFunc) (*storage.Store, *Handler, *httptest.Server) {
	return setupTestServerWithHandler(t, mockLLM, llmHandler)
}

func setupTestServerWithHandler(t *testing.T, mockLLM bool, llmHandler http.HandlerFunc) (*storage.Store, *Handler, *httptest.Server) {
	t.Helper()
	t.Setenv("LANGFUSE_ENABLED", "false")
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmURL := "https://api.deepseek.com"
	if mockLLM {
		if llmHandler == nil {
			llmHandler = goConcurrencyLLMMock(nil)
		}
		mock := httptest.NewServer(llmHandler)
		t.Cleanup(mock.Close)
		llmURL = mock.URL
	}

	h, err := NewHandler(store, llm.NewClient("test-key", llmURL))
	if err != nil {
		t.Fatal(err)
	}
	return store, h, httptest.NewServer(NewServer(h, nil, nil))
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

func postDomainBuildJob(t *testing.T, baseURL string, payload any) string {
	t.Helper()
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/domain/build", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("build accept status=%d body=%s", resp.StatusCode, string(b))
	}
	var accepted map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&accepted); err != nil {
		t.Fatal(err)
	}
	if accepted["status"] != "accepted" || accepted["jobId"] == "" {
		t.Fatalf("unexpected accept body: %+v", accepted)
	}
	return accepted["jobId"]
}

func pollBuildJob(t *testing.T, baseURL, jobID string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/domain/build/jobs/" + jobID)
		if err != nil {
			t.Fatal(err)
		}
		var job map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		status, _ := job["status"].(string)
		switch status {
		case storage.DomainBuildJobDone:
			return job
		case storage.DomainBuildJobFailed:
			msg, _ := job["error"].(string)
			if msg == "" {
				msg, _ = job["message"].(string)
			}
			t.Fatalf("建课任务失败: %s", msg)
		case storage.DomainBuildJobRunning:
			time.Sleep(150 * time.Millisecond)
		default:
			t.Fatalf("未知任务状态: %+v", job)
		}
	}
	t.Fatal("建课任务轮询超时")
	return nil
}

func buildDomainResult(t *testing.T, baseURL string, payload any) map[string]any {
	t.Helper()
	jobID := postDomainBuildJob(t, baseURL, payload)
	job := pollBuildJob(t, baseURL, jobID)
	raw, ok := job["result"]
	if !ok {
		t.Fatalf("任务完成但缺少 result: %+v", job)
	}
	result, ok := raw.(map[string]any)
	if !ok {
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &result); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func decodeBuildTreeFromResult(t *testing.T, result map[string]any) storage.KnowledgeTree {
	t.Helper()
	raw, err := json.Marshal(result["tree"])
	if err != nil {
		t.Fatal(err)
	}
	var tree storage.KnowledgeTree
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	return tree
}

func buildGoConcurrencyDomain(t *testing.T, baseURL string) storage.KnowledgeTree {
	t.Helper()
	result := buildDomainResult(t, baseURL, map[string]string{"name": "Go 并发"})
	tree := decodeBuildTreeFromResult(t, result)
	if tree.DomainID == "" {
		t.Fatal("缺少 domainId")
	}
	return tree
}

func TestBuildDomainAndTree(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)
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
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleRustTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"用户想学 Rust\",\"scopeBreadth\":\"moderate\"}"}}]}`))
	}
	_, _, ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()

	result := buildDomainResult(t, ts.URL, map[string]string{"name": "rust"})
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

	tree := buildGoConcurrencyDomain(t, ts.URL)

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
	ts := setupTestServer(t, true)
	defer ts.Close()

	buildGoConcurrencyDomain(t, ts.URL)

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
	ts := setupTestServer(t, true)
	defer ts.Close()

	payload := map[string]string{"name": "Go 并发"}
	result1 := buildDomainResult(t, ts.URL, payload)
	tree1 := decodeBuildTreeFromResult(t, result1)
	result2 := buildDomainResult(t, ts.URL, payload)
	tree2 := decodeBuildTreeFromResult(t, result2)

	if tree1.DomainID != tree2.DomainID {
		t.Fatalf("期望相同 domainId，得到 %s vs %s", tree1.DomainID, tree2.DomainID)
	}
}

func TestActiveSessionResume(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)

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
	smartMock := goConcurrencyLLMMock(func(w http.ResponseWriter, body string) bool {
		if strings.Contains(body, "exercise.json") || strings.Contains(body, "小练习") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"question\":\"1+1=?\",\"question_type\":\"short_answer\",\"answer_format\":\"text\",\"reinforced_concepts\":[]}"}}]}`))
			return true
		}
		return false
	})
	_, _, ts := setupTestServerWithHandler(t, true, smartMock)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)

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
	ts := setupTestServer(t, true)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)

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

func TestRegenerateDomainPreservesProgress(t *testing.T) {
	chdirToRepo(t)
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleRustTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"用户想学 Rust\",\"scopeBreadth\":\"moderate\"}"}}]}`))
	}
	store, _, ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()

	result := buildDomainResult(t, ts.URL, map[string]string{"name": "rust"})
	tree := decodeBuildTreeFromResult(t, result)
	if err := store.UpsertProgress(storage.UserProgress{
		UserID: storage.DefaultUserID, DomainID: tree.DomainID,
		NodeKey: "rust_basics", Layer: "entry", Status: "completed", Mastery: 1,
	}); err != nil {
		t.Fatal(err)
	}

	regBody, _ := json.Marshal(map[string]string{"confirmName": tree.DomainName})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/"+tree.DomainID+"/regenerate", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("regenerate status=%d body=%s", resp.StatusCode, string(b))
	}
	var regResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&regResult); err != nil {
		t.Fatal(err)
	}
	if kept, _ := regResult["progressKept"].(float64); kept < 1 {
		t.Fatalf("progressKept=%v", regResult["progressKept"])
	}
	newTree := decodeBuildTreeFromMap(t, regResult)
	if newTree.DomainID == tree.DomainID {
		t.Fatal("expected new domain id after regenerate")
	}
	list, err := store.ListProgress(storage.DefaultUserID, newTree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range list {
		if p.NodeKey == "rust_basics" && p.Status == "completed" {
			found = true
		}
	}
	if !found {
		t.Fatal("completed progress should exist on new domain")
	}
	if _, err := store.GetDomain(storage.DefaultUserID, tree.DomainID); err == nil {
		t.Fatal("old domain should be deleted")
	}
}

func TestRegenerateDomainReportsUnmigratedProgress(t *testing.T) {
	chdirToRepo(t)
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleRustTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"用户想学 Rust\",\"scopeBreadth\":\"moderate\"}"}}]}`))
	}
	store, _, ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()

	result := buildDomainResult(t, ts.URL, map[string]string{"name": "rust"})
	tree := decodeBuildTreeFromResult(t, result)
	if err := store.UpsertProgress(storage.UserProgress{
		UserID: storage.DefaultUserID, DomainID: tree.DomainID,
		NodeKey: "legacy_only_node", Layer: "entry", Status: "completed", Mastery: 1,
	}); err != nil {
		t.Fatal(err)
	}

	regBody, _ := json.Marshal(map[string]string{"confirmName": tree.DomainName})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/"+tree.DomainID+"/regenerate", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("regenerate status=%d body=%s", resp.StatusCode, string(b))
	}
	var regResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&regResult); err != nil {
		t.Fatal(err)
	}
	if kept, _ := regResult["progressKept"].(float64); kept != 0 {
		t.Fatalf("progressKept=%v want 0", kept)
	}
	if skipped, _ := regResult["progressSkipped"].(float64); skipped != 1 {
		t.Fatalf("progressSkipped=%v want 1", skipped)
	}
	msg, _ := regResult["message"].(string)
	if !strings.Contains(msg, "未能自动迁移") {
		t.Fatalf("message=%q", msg)
	}
}

func TestRegenerateDomainWrongConfirmName(t *testing.T) {
	chdirToRepo(t)
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleRustTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"\",\"scopeBreadth\":\"moderate\"}"}}]}`))
	}
	_, _, ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()
	result := buildDomainResult(t, ts.URL, map[string]string{"name": "rust"})
	tree := decodeBuildTreeFromResult(t, result)
	body, _ := json.Marshal(map[string]string{"confirmName": "wrong"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/"+tree.DomainID+"/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func decodeBuildTreeFromMap(t *testing.T, result map[string]any) storage.KnowledgeTree {
	t.Helper()
	raw, err := json.Marshal(result["tree"])
	if err != nil {
		t.Fatal(err)
	}
	var tree storage.KnowledgeTree
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	return tree
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

func isTreeBuildLLMRequest(body string) bool {
	if strings.Contains(body, "知识树设计师") {
		return true
	}
	// json.Marshal 会将中文转义为 \uXXXX，需同时匹配
	if strings.Contains(body, `\u77e5\u8bc6\u6811\u8bbe\u8ba1\u5e08`) {
		return true
	}
	return strings.Contains(body, "exercise_ideas")
}

func goConcurrencyLLMMock(extra func(w http.ResponseWriter, body string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		w.Header().Set("Content-Type", "application/json")
		if extra != nil && extra(w, body) {
			return
		}
		if strings.Contains(body, "材料片段") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"points\":[\"要点\"],\"concepts\":[\"概念\"]}"}}]}`))
			return
		}
		if strings.Contains(body, "各片段要点汇总") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"title\":\"导入主题\",\"sections\":[{\"heading\":\"章节\",\"points\":[\"要点\"],\"concepts\":[\"概念\"]}],\"scopeBreadth\":\"moderate\"}"}}]}`))
			return
		}
		if strings.Contains(body, "扩展设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleExtendTreeJSON) + `}}]}`))
			return
		}
		// Coach 结构化任务（出题/批改/掌握度/画像）
		if strings.Contains(body, "exercise.json") ||
			strings.Contains(body, "grade.json") ||
			strings.Contains(body, "mastery_check.json") ||
			strings.Contains(body, "profile_refresh.json") ||
			strings.Contains(body, "profile_init.json") ||
			strings.Contains(body, "设置页画像合并") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"这是测试讲解。回复「开始练习」继续。"}}]}`))
			return
		}
		// 动态建树（ChatJSON + json_object，且非 Coach schema）
		if isTreeBuildLLMRequest(body) || strings.Contains(body, `"type":"json_object"`) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleGoRootTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"这是测试讲解。回复「开始练习」继续。"}}]}`))
	}
}

const sampleGoRootTreeJSON = `{
  "domain": "Go 语言",
  "slug": "go",
  "description": "系统学习 Go，能独立编写可靠的后端服务",
  "modules": [
    { "key": "go_basics", "label": "语言基础", "goal": "语法与类型", "nodes": ["go_syntax", "go_types", "go_functions"] },
    { "key": "go_packages_io", "label": "包与 IO", "goal": "组织代码与读写", "nodes": ["go_packages", "go_io", "go_json"] },
    { "key": "go_quality", "label": "工程质量", "goal": "错误处理与测试", "nodes": ["go_errors", "go_testing", "go_tools"] },
    { "key": "go_advanced", "label": "进阶主题", "goal": "性能与工具链", "nodes": ["go_perf", "go_modules_adv", "go_cgo"] }
  ],
  "layers": {
    "entry": {
      "label": "入门", "time": "约 4～6 小时", "goal": "掌握 Go 基础语法，能读懂常见代码",
      "nodes": [{"key": "go_syntax", "title": "基础语法"}, {"key": "go_types", "title": "类型系统"}, {"key": "go_functions", "title": "函数"}, {"key": "go_packages", "title": "包与模块"}]
    },
    "intermediate": {
      "label": "熟悉", "time": "约 10～14 小时", "goal": "能编写日常 Go 程序",
      "nodes": [{"key": "go_errors", "title": "错误处理"}, {"key": "go_testing", "title": "测试"}, {"key": "go_io", "title": "IO"}, {"key": "go_json", "title": "JSON 处理"}]
    },
    "advanced": {
      "label": "精通", "time": "约 18～24 小时", "goal": "理解性能与工具链",
      "nodes": [{"key": "go_perf", "title": "性能优化"}, {"key": "go_tools", "title": "工具链"}, {"key": "go_modules_adv", "title": "模块进阶"}, {"key": "go_cgo", "title": "CGO 基础"}]
    }
  },
  "nodes": [
    {"key":"go_syntax","node":"基础语法","layer":"入门","core_concepts":["变量声明"],"common_mistakes":[":= 误用"],"boundaries":["不讲并发"],"exercise_ideas":["解释 := 与 var"]},
    {"key":"go_types","node":"类型系统","layer":"入门","core_concepts":["struct"],"common_mistakes":["值拷贝"],"boundaries":["不讲反射"],"exercise_ideas":["定义 struct"]},
    {"key":"go_functions","node":"函数","layer":"入门","core_concepts":["多返回值"],"common_mistakes":["忽略 error"],"boundaries":["不讲泛型深入"],"exercise_ideas":["写带 error 的函数"]},
    {"key":"go_packages","node":"包与模块","layer":"入门","core_concepts":["package"],"common_mistakes":["循环导入"],"boundaries":["不讲 vendoring"],"exercise_ideas":["拆分 package"]},
    {"key":"go_errors","node":"错误处理","layer":"熟悉","core_concepts":["error 接口"],"common_mistakes":["吞掉 error"],"boundaries":["不讲 panic 恢复深入"],"exercise_ideas":["包装 error"]},
    {"key":"go_testing","node":"测试","layer":"熟悉","core_concepts":["testing 包"],"common_mistakes":["不测边界"],"boundaries":["不讲 benchmark 源码"],"exercise_ideas":["写 table test"]},
    {"key":"go_io","node":"IO","layer":"熟悉","core_concepts":["Reader/Writer"],"common_mistakes":["未 Close"],"boundaries":["不讲 net 包"],"exercise_ideas":["读文件"]},
    {"key":"go_json","node":"JSON","layer":"熟悉","core_concepts":["encoding/json"],"common_mistakes":["tag 写错"],"boundaries":["不讲 protobuf"],"exercise_ideas":["序列化 struct"]},
    {"key":"go_perf","node":"性能优化","layer":"精通","core_concepts":["pprof"],"common_mistakes":["过早优化"],"boundaries":["不讲汇编"],"exercise_ideas":["读 pprof 报告"]},
    {"key":"go_tools","node":"工具链","layer":"精通","core_concepts":["go mod"],"common_mistakes":["版本冲突"],"boundaries":["不讲私有 proxy"],"exercise_ideas":["解释 go mod tidy"]},
    {"key":"go_modules_adv","node":"模块进阶","layer":"精通","core_concepts":["replace 指令"],"common_mistakes":["replace 泄漏"],"boundaries":["不讲 workspace 全部特性"],"exercise_ideas":["本地 replace"]},
    {"key":"go_cgo","node":"CGO 基础","layer":"精通","core_concepts":["C 互操作"],"common_mistakes":["跨边界内存"],"boundaries":["不讲复杂 C 库"],"exercise_ideas":["解释 //export"]}
  ]
}`

const sampleRustTreeJSON = `{
  "domain": "Rust",
  "slug": "rust",
  "description": "系统学习 Rust，能独立开发可靠的后端服务",
  "modules": [
    { "key": "syntax_ownership", "label": "语法与所有权", "goal": "掌握基础语法与内存模型", "nodes": ["rust_basics", "ownership", "borrowing"] },
    { "key": "types_abstraction", "label": "类型与抽象", "goal": "结构体、枚举与 trait", "nodes": ["structs", "enums", "traits"] },
    { "key": "advanced_topics", "label": "进阶机制", "goal": "生命周期与异步", "nodes": ["lifetimes", "async_rust"] }
  ],
  "layers": {
    "entry": {
      "label": "入门", "time": "约 4～6 小时", "goal": "掌握基础语法与所有权，能读懂常见 Rust 代码并建立语言知识框架",
      "nodes": [{"key": "rust_basics", "title": "Rust 基础"}, {"key": "ownership", "title": "所有权"}, {"key": "borrowing", "title": "借用与引用"}]
    },
    "intermediate": {
      "label": "熟悉", "time": "约 12～18 小时", "goal": "能编写结构化的 Rust 程序，处理日常开发中的大多数场景",
      "nodes": [{"key": "structs", "title": "结构体"}, {"key": "enums", "title": "枚举与模式匹配"}, {"key": "traits", "title": "Trait 与泛型"}]
    },
    "advanced": {
      "label": "精通", "time": "约 20～30 小时", "goal": "能处理生命周期、异步与性能等复杂问题，应对绝大多数工程场景",
      "nodes": [{"key": "lifetimes", "title": "生命周期"}, {"key": "async_rust", "title": "异步 Rust"}]
    }
  },
  "nodes": [
    {"key":"rust_basics","node":"Rust 基础","layer":"入门","core_concepts":["变量与类型"],"common_mistakes":["混淆 mut"],"boundaries":["不讲 async"],"exercise_ideas":["解释 mut 的作用"]},
    {"key":"ownership","node":"所有权","layer":"入门","core_concepts":["所有权规则"],"common_mistakes":["双重可变借用"],"boundaries":["不讲生命周期细节"],"exercise_ideas":["判断能否编译"]},
    {"key":"borrowing","node":"借用与引用","layer":"入门","core_concepts":["不可变/可变借用"],"common_mistakes":["悬垂引用"],"boundaries":["不讲生命周期标注"],"exercise_ideas":["修复借用错误"]},
    {"key":"structs","node":"结构体","layer":"熟悉","core_concepts":["struct 定义"],"common_mistakes":["字段可见性"],"boundaries":["不讲 trait 对象"],"exercise_ideas":["定义一个 Point"]},
    {"key":"enums","node":"枚举","layer":"熟悉","core_concepts":["enum 与 match"],"common_mistakes":["遗漏分支"],"boundaries":["不讲 GADT"],"exercise_ideas":["用 match 处理 Option"]},
    {"key":"traits","node":"Trait 与泛型","layer":"熟悉","core_concepts":["trait 定义与实现"],"common_mistakes":["孤儿规则"],"boundaries":["不讲关联类型深入"],"exercise_ideas":["为类型实现 Display"]},
    {"key":"lifetimes","node":"生命周期","layer":"精通","core_concepts":["生命周期标注"],"common_mistakes":["不必要的 'a"],"boundaries":["不讲 HRTB"],"exercise_ideas":["标注函数签名"]},
    {"key":"async_rust","node":"异步 Rust","layer":"精通","core_concepts":["async/await"],"common_mistakes":["阻塞 runtime"],"boundaries":["不讲 tokio 源码"],"exercise_ideas":["写一个 async fn"]}
  ]
}`

func TestRefineUserProfileAPI(t *testing.T) {
	chdirToRepo(t)
	store, _, ts := setupTestServerStore(t, true, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(body, "设置页画像合并") || strings.Contains(body, "【用户补充】") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"【背景】工程师\\n【进展】在学 Go\"}"}}]}`))
			return
		}
		goConcurrencyLLMMock(nil)(w, r)
	})
	defer ts.Close()

	user, err := store.CreateUser("画像测试")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpdateUserProfileSummary(user.ID, "旧画像")

	body, _ := json.Marshal(map[string]string{"supplement": "最近在学 Go"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/users/profile/refine", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
	var out storage.User
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.ProfileSummary, "Go") {
		t.Fatalf("profile=%q", out.ProfileSummary)
	}
}

const sampleExtendTreeJSON = `{
  "layers": {
    "advanced": {
      "label": "精通", "time": "约 6 小时", "goal": "进阶实战",
      "nodes": [{"key":"go_capstone","title":"Go 实战项目"}]
    }
  },
  "nodes": [{
    "key":"go_capstone","node":"Go 实战项目","layer":"精通",
    "core_concepts":["项目结构"],"common_mistakes":["过度设计"],"boundaries":["不讲底层"],"exercise_ideas":["规划模块"]
  }],
  "modules": [{
    "key":"go_capstone_mod","label":"实战进阶","nodes":["go_capstone"]
  }]
}`

func TestBuildDomainFromSourceURL(t *testing.T) {
	t.Setenv("REGULUS_INGEST_ALLOW_PRIVATE", "1")
	chdirToRepo(t)
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article><h1>导入主题</h1><p>测试正文内容足够长。</p></article></body></html>`))
	}))
	defer page.Close()

	ts := setupTestServer(t, true)
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"url": page.URL, "name": "导入主题"})
	resp, err := http.Post(ts.URL+"/api/domain/build/from-source", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
	var accepted map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&accepted); err != nil {
		t.Fatal(err)
	}
	job := pollBuildJob(t, ts.URL, accepted["jobId"])
	resultRaw, ok := job["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %+v", job)
	}
	if resultRaw["status"] != "ready" {
		t.Fatalf("result=%+v", resultRaw)
	}
}

func TestExtendDomainAPI(t *testing.T) {
	chdirToRepo(t)
	t.Setenv("REGULUS_TREE_CRITIQUE", "0")
	store, _, ts := setupTestServerStore(t, true, nil)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)
	keys := domain.CollectTreeNodeKeys(&tree)
	need := (len(keys)*8 + 9) / 10
	if need > len(keys) {
		need = len(keys)
	}
	for _, key := range keys[:need] {
		if err := store.UpsertProgress(storage.UserProgress{
			UserID: storage.DefaultUserID, DomainID: tree.DomainID, NodeKey: key,
			Layer: "entry", Status: "completed", Mastery: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}

	eligResp, err := http.Get(ts.URL + "/api/domain/" + tree.DomainID + "/extend/eligibility")
	if err != nil {
		t.Fatal(err)
	}
	var elig map[string]any
	_ = json.NewDecoder(eligResp.Body).Decode(&elig)
	eligResp.Body.Close()

	extBody, _ := json.Marshal(map[string]any{"confirm": true, "goal": "进阶实战"})
	extResp, err := http.Post(ts.URL+"/api/domain/"+tree.DomainID+"/extend", "application/json", bytes.NewReader(extBody))
	if err != nil {
		t.Fatal(err)
	}
	defer extResp.Body.Close()
	if extResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(extResp.Body)
		t.Fatalf("extend status=%d body=%s elig=%+v", extResp.StatusCode, string(b), elig)
	}
	var extOut map[string]any
	if err := json.NewDecoder(extResp.Body).Decode(&extOut); err != nil {
		t.Fatal(err)
	}
	if extOut["treeVersion"] != float64(2) {
		t.Fatalf("treeVersion=%v", extOut["treeVersion"])
	}
}

func TestExportDomainReturnsZip(t *testing.T) {
	chdirToRepo(t)
	ts := setupTestServer(t, true)
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)

	resp, err := http.Get(ts.URL + "/api/domain/" + tree.DomainID + "/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("export status=%d body=%s", resp.StatusCode, string(b))
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/zip") {
		t.Fatalf("Content-Type 应为 application/zip，得到 %s", ct)
	}
	disp := resp.Header.Get("Content-Disposition")
	if !strings.Contains(disp, "-skill.zip") {
		t.Fatalf("Content-Disposition 应含 -skill.zip，得到 %s", disp)
	}
}

func TestExportDomainLLMFailureStillSucceeds(t *testing.T) {
	chdirToRepo(t)
	// 建树用正常 mock；description 补全请求（含「30字以内」）让 LLM 返回 500，导出仍应成功
	extra := func(w http.ResponseWriter, body string) bool {
		if strings.Contains(body, "30字以内") {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		return false
	}
	_, _, ts := setupTestServerWithHandler(t, true, goConcurrencyLLMMock(extra))
	defer ts.Close()

	tree := buildGoConcurrencyDomain(t, ts.URL)

	resp, err := http.Get(ts.URL + "/api/domain/" + tree.DomainID + "/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("LLM 失败时导出仍应成功，status=%d body=%s", resp.StatusCode, string(b))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/zip") {
		t.Fatalf("应返回 zip，得到 Content-Type=%s", ct)
	}
}
