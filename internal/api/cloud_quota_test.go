package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/cloud"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func setupCloudQuotaTest(t *testing.T, msgLimit, buildLimit int) (*httptest.Server, *storage.Store) {
	t.Helper()
	t.Setenv("LANGFUSE_ENABLED", "false")
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmClient := llm.NewClient("test-key", "https://api.deepseek.com")
	cfg := cloud.Config{
		Deployment:             cloud.DeploymentCloud,
		QuotaDailyMessages:     msgLimit,
		QuotaDailyBuilds:       buildLimit,
		EncryptionKey:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MaxBuildJobsGlobal:     5,
		RateLimitPerIP:         120,
		MaxUsersCreatePerIPDay: 5,
	}
	cloudSvc := cloud.NewService(cfg, store, llmClient)
	h, err := NewHandler(store, llmClient, cloudSvc)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(NewServer(h, nil, nil)), store
}

func createTestUser(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"displayName": "配额测试"})
	resp, err := http.Post(ts.URL+"/api/users", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create user: %d", resp.StatusCode)
	}
	var u struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func TestCloudBuildQuotaExceeded(t *testing.T) {
	ts, store := setupCloudQuotaTest(t, 30, 1)
	defer ts.Close()
	uid := createTestUser(t, ts)

	if err := store.IncrementLLMBuildCount(uid, storage.TodayUTC()); err != nil {
		t.Fatal(err)
	}

	reqBody, _ := json.Marshal(map[string]string{"name": "自定义主题A"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/build", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", uid)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["code"] != "build_quota_exceeded" {
		t.Fatalf("code=%v", out["code"])
	}
}

func TestCloudSkillBuildExemptFromQuota(t *testing.T) {
	ts, store := setupCloudQuotaTest(t, 30, 0)
	defer ts.Close()
	uid := createTestUser(t, ts)

	reqBody, _ := json.Marshal(map[string]string{"name": "Go 并发"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/build", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", uid)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("skill build: want 202, got %d", resp.StatusCode)
	}
	usage, err := store.GetLLMUsageDaily(uid, storage.TodayUTC())
	if err != nil {
		t.Fatal(err)
	}
	if usage.BuildCount != 0 {
		t.Fatalf("build_count=%d, want 0 for skill exempt", usage.BuildCount)
	}
}

func TestCloudCreateUserIPLimit(t *testing.T) {
	ts, _ := setupCloudQuotaTest(t, 30, 3)
	defer ts.Close()

	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{"displayName": "用户"})
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.10")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create %d: want 200, got %d", i, resp.StatusCode)
		}
	}

	body, _ := json.Marshal(map[string]string{"displayName": "超限"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", resp.StatusCode)
	}
}

func TestCloudQuotaStatusIncludesBuild(t *testing.T) {
	ts, _ := setupCloudQuotaTest(t, 30, 3)
	defer ts.Close()
	uid := createTestUser(t, ts)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/user/llm-quota", nil)
	req.Header.Set("X-User-Id", uid)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if int(out["buildLimit"].(float64)) != 3 {
		t.Fatalf("buildLimit=%v", out["buildLimit"])
	}
}
