package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyLLMSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("LLM_PROVIDER=deepseek\nLLM_API_KEY=sk-old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	SetLLMEnvFile(path)
	t.Cleanup(func() { SetLLMEnvFile(DefaultEnvPath) })
	t.Setenv("LLM_API_KEY", "sk-old")

	if err := ApplyLLMSettings(LLMSettingsPayload{
		Provider: "openai",
		Model:    "gpt-4o",
	}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("LLM_PROVIDER") != "openai" {
		t.Fatalf("provider=%q", os.Getenv("LLM_PROVIDER"))
	}
	if os.Getenv("LLM_MODEL") != "gpt-4o" {
		t.Fatalf("model=%q", os.Getenv("LLM_MODEL"))
	}
	if os.Getenv("LLM_API_KEY") != "sk-old" {
		t.Fatalf("key should be preserved, got %q", os.Getenv("LLM_API_KEY"))
	}
}
