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

func setupCloudTestServer(t *testing.T) *httptest.Server {
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
		Deployment:         cloud.DeploymentCloud,
		QuotaDailyMessages: 20,
		EncryptionKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MaxBuildJobsGlobal: 3,
		RateLimitPerIP:     60,
	}
	cloudSvc := cloud.NewService(cfg, store, llmClient)
	h, err := NewHandler(store, llmClient, cloudSvc)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(NewServer(h, nil, nil))
}

func TestCloudAnonymousCreateUser(t *testing.T) {
	ts := setupCloudTestServer(t)
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"displayName": "小明"})
	resp, err := http.Post(ts.URL+"/api/users", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/users status = %d, want 200", resp.StatusCode)
	}
}

func TestCloudAnonymousListUsers(t *testing.T) {
	ts := setupCloudTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/users status = %d, want 200", resp.StatusCode)
	}
}

func TestCloudRequiresUserForProtectedAPI(t *testing.T) {
	ts := setupCloudTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/domains")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/domains status = %d, want 401", resp.StatusCode)
	}
}
