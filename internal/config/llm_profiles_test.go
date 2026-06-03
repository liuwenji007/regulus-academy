package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadLLMProfiles(t *testing.T) {
	dir := t.TempDir()
	SetLLMProfilesFile(filepath.Join(dir, "llm-profiles.json"))
	SetLLMEnvFile(filepath.Join(dir, ".env"))
	t.Cleanup(func() {
		SetLLMProfilesFile("")
		SetLLMEnvFile(DefaultEnvPath)
	})
	t.Setenv("LLM_API_KEY", "sk-test")

	state := LLMProfilesState{
		ActiveID: "a1",
		Profiles: []LLMProfile{
			{ID: "a1", Name: "DeepSeek 主力", Provider: "deepseek", Model: "deepseek-chat"},
			{ID: "a2", Name: "自定义", Provider: "custom", BaseURL: "https://api.example.com", Model: "hy3-preview"},
		},
	}
	if err := SaveLLMProfiles(state); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadLLMProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("profiles=%d", len(loaded.Profiles))
	}
	if loaded.ActiveID != "a1" {
		t.Fatalf("active=%q", loaded.ActiveID)
	}
}

func TestApplyActiveLLMProfile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("LLM_API_KEY=sk-old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	SetLLMEnvFile(envPath)
	SetLLMProfilesFile(filepath.Join(dir, "llm-profiles.json"))
	t.Cleanup(func() {
		SetLLMEnvFile(DefaultEnvPath)
		SetLLMProfilesFile("")
	})
	t.Setenv("LLM_API_KEY", "sk-old")

	state := LLMProfilesState{
		ActiveID: "c1",
		Profiles: []LLMProfile{{
			ID: "c1", Name: "Test", Provider: "custom",
			BaseURL: "https://api.example.com", Model: "hy3-preview",
		}},
	}
	if err := SaveLLMProfiles(state); err != nil {
		t.Fatal(err)
	}
	if err := ApplyActiveLLMProfile(state); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("LLM_PROVIDER") != "custom" {
		t.Fatalf("provider=%q", os.Getenv("LLM_PROVIDER"))
	}
	if os.Getenv("LLM_MODEL") != "hy3-preview" {
		t.Fatalf("model=%q", os.Getenv("LLM_MODEL"))
	}
}

func TestMergeProfileAPIKeysFromExisting(t *testing.T) {
	existing := LLMProfilesState{
		ActiveID: "a1",
		Profiles: []LLMProfile{
			{ID: "a1", Name: "A", Provider: "deepseek", Model: "deepseek-chat", APIKey: "sk-a"},
			{ID: "a2", Name: "B", Provider: "custom", BaseURL: "https://api.example.com", Model: "m1", APIKey: "sk-b"},
		},
	}
	incoming := LLMProfilesState{
		ActiveID: "a2",
		Profiles: []LLMProfile{
			{ID: "a1", Name: "A renamed", Provider: "deepseek", Model: "deepseek-chat"},
			{ID: "a2", Name: "B", Provider: "custom", BaseURL: "https://api.example.com", Model: "m1"},
		},
	}
	merged := MergeProfileAPIKeysFromExisting(existing, incoming)
	if merged.Profiles[0].APIKey != "sk-a" {
		t.Fatalf("a1 key=%q", merged.Profiles[0].APIKey)
	}
	if merged.Profiles[1].APIKey != "sk-b" {
		t.Fatalf("a2 key=%q", merged.Profiles[1].APIKey)
	}
}

func TestApplyActiveLLMProfileSwitchesAPIKey(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("LLM_API_KEY=sk-global\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	SetLLMEnvFile(envPath)
	SetLLMProfilesFile(filepath.Join(dir, "llm-profiles.json"))
	t.Cleanup(func() {
		SetLLMEnvFile(DefaultEnvPath)
		SetLLMProfilesFile("")
	})
	t.Setenv("LLM_API_KEY", "sk-global")

	state := LLMProfilesState{
		ActiveID:     "a1",
		GlobalAPIKey: "sk-global",
		Profiles: []LLMProfile{
			{ID: "a1", Name: "Global", Provider: "deepseek", Model: "deepseek-chat"},
			{ID: "a2", Name: "Custom", Provider: "custom", BaseURL: "https://api.example.com", Model: "m1", APIKey: "sk-dedicated"},
		},
	}
	if err := SaveLLMProfiles(state); err != nil {
		t.Fatal(err)
	}

	state.ActiveID = "a2"
	if err := ApplyActiveLLMProfile(state); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("LLM_API_KEY") != "sk-dedicated" {
		t.Fatalf("active a2 key=%q", os.Getenv("LLM_API_KEY"))
	}

	state.ActiveID = "a1"
	if err := ApplyActiveLLMProfile(state); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("LLM_API_KEY") != "sk-global" {
		t.Fatalf("active a1 should restore global key, got %q", os.Getenv("LLM_API_KEY"))
	}
}
