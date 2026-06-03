package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestUpdateLLMConfig(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("LLM_PROVIDER=deepseek\nLLM_API_KEY=sk-test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config.SetLLMEnvFile(envPath)
	config.SetLLMProfilesFile(filepath.Join(dir, "llm-profiles.json"))
	t.Cleanup(func() {
		config.SetLLMEnvFile(config.DefaultEnvPath)
		config.SetLLMProfilesFile("")
	})

	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	h, err := NewHandler(store, llm.NewClient("sk-test", "https://api.deepseek.com"))
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]string{
		"provider": "openai",
		"model":    "gpt-4o-mini",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/llm/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.updateLLMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if h.llmClient().Model() != "gpt-4o-mini" {
		t.Fatalf("model=%s", h.llmClient().Model())
	}
}

func TestReloadLLMConcurrent(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("LLM_PROVIDER=deepseek\nLLM_API_KEY=sk-test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config.SetLLMEnvFile(envPath)
	t.Cleanup(func() { config.SetLLMEnvFile(config.DefaultEnvPath) })

	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	h, err := NewHandler(store, llm.NewClient("sk-test", "https://api.deepseek.com"))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = h.llmClient().Model()
				_ = h.llmClient().Configured()
				_ = h.llmClient().Name()
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if err := h.reloadLLM(); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestUpdateLLMProfilesPreservesAPIKeys(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("LLM_PROVIDER=deepseek\nLLM_API_KEY=sk-global\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config.SetLLMEnvFile(envPath)
	profilesPath := filepath.Join(dir, "llm-profiles.json")
	config.SetLLMProfilesFile(profilesPath)
	t.Cleanup(func() {
		config.SetLLMEnvFile(config.DefaultEnvPath)
		config.SetLLMProfilesFile("")
	})

	initial := config.LLMProfilesState{
		ActiveID:     "a1",
		GlobalAPIKey: "sk-global",
		Profiles: []config.LLMProfile{
			{ID: "a1", Name: "Global", Provider: "deepseek", Model: "deepseek-chat"},
			{ID: "a2", Name: "Custom", Provider: "custom", BaseURL: "https://api.example.com", Model: "m1", APIKey: "sk-dedicated"},
		},
	}
	if err := config.SaveLLMProfiles(initial); err != nil {
		t.Fatal(err)
	}

	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	h, err := NewHandler(store, llm.NewClient("sk-global", "https://api.deepseek.com"))
	if err != nil {
		t.Fatal(err)
	}

	// 模拟 Web 保存：不传 apiKey，切换当前使用为 a2
	body, _ := json.Marshal(map[string]any{
		"activeId": "a2",
		"profiles": []map[string]string{
			{"id": "a1", "name": "Global", "provider": "deepseek", "model": "deepseek-chat"},
			{"id": "a2", "name": "Custom", "provider": "custom", "baseUrl": "https://api.example.com", "model": "m1"},
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/llm/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.updateLLMProfiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	loaded, err := config.LoadLLMProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range loaded.Profiles {
		if p.ID == "a2" && p.APIKey != "sk-dedicated" {
			t.Fatalf("a2 key=%q want sk-dedicated", p.APIKey)
		}
	}
	if os.Getenv("LLM_API_KEY") != "sk-dedicated" {
		t.Fatalf(".env key=%q want sk-dedicated", os.Getenv("LLM_API_KEY"))
	}
}
