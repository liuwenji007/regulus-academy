package llm

import (
	"os"
	"testing"
)

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")

	cfg := ConfigFromEnv()
	if cfg.Provider != "deepseek" {
		t.Fatalf("provider=%s", cfg.Provider)
	}
	if cfg.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("baseURL=%s", cfg.BaseURL)
	}
	if cfg.Model != "deepseek-chat" {
		t.Fatalf("model=%s", cfg.Model)
	}
}

func TestConfigFromEnvOpenAI(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_API_KEY", "sk-test")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := ConfigFromEnv()
	if cfg.BaseURL != "https://api.openai.com" {
		t.Fatalf("baseURL=%s", cfg.BaseURL)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Fatalf("model=%s", cfg.Model)
	}
}

func TestConfigFromEnvLegacyDeepSeek(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "sk-legacy")
	t.Setenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")

	cfg := ConfigFromEnv()
	if cfg.APIKey != "sk-legacy" {
		t.Fatalf("apiKey=%s", cfg.APIKey)
	}
	if cfg.Provider != "deepseek" {
		t.Fatalf("provider=%s", cfg.Provider)
	}
}

func TestOllamaConfiguredWithoutKey(t *testing.T) {
	p := NewFromConfig(OpenAIConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "llama3",
	})
	if !p.Configured() {
		t.Fatal("ollama 无 Key 也应视为已配置")
	}
}

func TestCustomRequiresURL(t *testing.T) {
	err := ValidateConfig(OpenAIConfig{Provider: "custom", APIKey: "k"})
	if err == nil {
		t.Fatal("期望 custom 缺 baseURL 时报错")
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://api.deepseek.com":                              "https://api.deepseek.com",
		"https://api.deepseek.com/":                             "https://api.deepseek.com",
		"https://tokenhub.tencentmaas.com/v1/chat/completions":  "https://tokenhub.tencentmaas.com",
		"https://tokenhub.tencentmaas.com/v1/chat/completions/": "https://tokenhub.tencentmaas.com",
	}
	for in, want := range cases {
		if got := normalizeBaseURL(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}

func TestMain(m *testing.M) {
	// 避免测试间环境变量泄漏
	code := m.Run()
	os.Exit(code)
}
