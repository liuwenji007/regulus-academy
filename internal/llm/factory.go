package llm

import (
	"fmt"
	"os"
	"strings"
)

// ConfigFromEnv 从环境变量解析 LLM 配置
func ConfigFromEnv() OpenAIConfig {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))

	// 兼容旧 DeepSeek 环境变量
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL"))
	}
	if provider == "" {
		if os.Getenv("DEEPSEEK_API_KEY") != "" || apiKey != "" {
			provider = "deepseek"
		} else {
			provider = "deepseek"
		}
	}

	preset, ok := GetPreset(provider)
	if !ok {
		preset = Preset{Name: provider}
	}

	if baseURL == "" {
		baseURL = preset.BaseURL
	}
	if model == "" {
		model = preset.Model
	}

	if provider == "custom" {
		if baseURL == "" || model == "" {
			// 允许启动，但调用时会因缺配置失败
			return OpenAIConfig{Provider: provider, APIKey: apiKey, BaseURL: baseURL, Model: model}
		}
	}

	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}

	return OpenAIConfig{
		Provider:    provider,
		APIKey:      apiKey,
		BaseURL:     normalizeBaseURL(baseURL),
		Model:       model,
		HTTPTimeout: HTTPTimeoutFromEnv(),
	}
}

// normalizeBaseURL 去掉末尾斜杠；若用户粘贴了完整 completions 地址则只保留 host 前缀
func normalizeBaseURL(baseURL string) string {
	u := strings.TrimSpace(baseURL)
	u = strings.TrimSuffix(u, "/")
	if strings.HasSuffix(u, "/v1/chat/completions") {
		u = strings.TrimSuffix(u, "/v1/chat/completions")
		u = strings.TrimSuffix(u, "/")
	}
	return u
}

// NewFromConfig 根据配置创建 Provider
func NewFromConfig(cfg OpenAIConfig) Provider {
	return NewOpenAI(cfg)
}

// NewFromEnv 从环境变量创建 Provider
func NewFromEnv() Provider {
	return NewFromConfig(ConfigFromEnv())
}

// ValidateConfig 校验 custom 等需完整配置的 provider
func ValidateConfig(cfg OpenAIConfig) error {
	if cfg.Provider == "custom" {
		if cfg.BaseURL == "" {
			return fmt.Errorf("LLM_PROVIDER=custom 时必须设置 LLM_BASE_URL")
		}
		if cfg.Model == "" {
			return fmt.Errorf("LLM_PROVIDER=custom 时必须设置 LLM_MODEL")
		}
	}
	if cfg.Provider != "ollama" && cfg.APIKey == "" {
		return fmt.Errorf("未配置 LLM_API_KEY（或 DEEPSEEK_API_KEY）")
	}
	return nil
}
