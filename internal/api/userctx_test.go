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
		Deployment:             cloud.DeploymentCloud,
		QuotaDailyMessages:     30,
		QuotaDailyBuilds:       3,
		EncryptionKey:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MaxBuildJobsGlobal:     5,
		RateLimitPerIP:         60,
		MaxUsersCreatePerIPDay: 5,
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

func TestCloudAdminAPIWithoutUserID(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "test-admin-secret")
	ts := setupCloudTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-admin-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/admin/stats with Bearer token status = %d, want 200", resp.StatusCode)
	}
}

func TestCloudAdminAPIRejectsMissingBearer(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "test-admin-secret")
	ts := setupCloudTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/admin/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/admin/stats without Bearer status = %d, want 401", resp.StatusCode)
	}
}
