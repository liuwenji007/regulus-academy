package config

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

// LLMSettingsPayload Web 端提交的 LLM 配置
type LLMSettingsPayload struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
}

// LLMSettingsView GET 返回的可编辑视图（密钥脱敏）
type LLMSettingsView struct {
	Provider    string `json:"provider"`
	APIKeySet   bool   `json:"apiKeySet"`
	BaseURL     string `json:"baseUrl"`
	Model       string `json:"model"`
	DisplayName string `json:"displayName"`
}

// LLMSettingsViewFromEnv 从当前环境变量构建可编辑视图
func LLMSettingsViewFromEnv() LLMSettingsView {
	cfg := llm.ConfigFromEnv()
	display := cfg.Provider
	if p, ok := llm.GetPreset(cfg.Provider); ok && p.Name != "" {
		display = p.Name
	}
	return LLMSettingsView{
		Provider:    cfg.Provider,
		APIKeySet:   strings.TrimSpace(cfg.APIKey) != "" || cfg.Provider == "ollama",
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		DisplayName: display,
	}
}

// ResolvedLLMConfig 根据提交项解析可探测/调用的 LLM 配置（不写 .env；API Key 留空则沿用当前环境）
func ResolvedLLMConfig(p LLMSettingsPayload) (llm.OpenAIConfig, error) {
	current := llm.ConfigFromEnv()

	provider := strings.ToLower(strings.TrimSpace(p.Provider))
	if provider == "" {
		provider = "deepseek"
	}
	if _, ok := llm.GetPreset(provider); !ok && provider != "custom" {
		return llm.OpenAIConfig{}, fmt.Errorf("不支持的 LLM_PROVIDER: %s", provider)
	}

	baseURL := strings.TrimSpace(p.BaseURL)
	model := strings.TrimSpace(p.Model)
	if provider != "custom" {
		if preset, ok := llm.GetPreset(provider); ok {
			if baseURL == "" {
				baseURL = preset.BaseURL
			}
			if model == "" {
				model = preset.Model
			}
		}
	}
	if provider == "custom" {
		if baseURL == "" {
			return llm.OpenAIConfig{}, fmt.Errorf("自定义提供商须填写 Base URL")
		}
		if model == "" {
			return llm.OpenAIConfig{}, fmt.Errorf("自定义提供商须填写模型名称")
		}
	}

	apiKey := strings.TrimSpace(p.APIKey)
	if apiKey == "" {
		if state, err := LoadLLMProfiles(); err == nil {
			apiKey = strings.TrimSpace(state.GlobalAPIKey)
		}
	}
	if apiKey == "" {
		apiKey = current.APIKey
	}

	cfg := llm.OpenAIConfig{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
	}
	if err := llm.ValidateConfig(cfg); err != nil {
		return llm.OpenAIConfig{}, err
	}
	return cfg, nil
}

// ApplyLLMSettings 写入 .env 并更新进程环境变量（API Key 留空表示不修改）
func ApplyLLMSettings(p LLMSettingsPayload) error {
	return applyLLMSettings(p, false)
}

func applyLLMSettings(p LLMSettingsPayload, forceAPIKey bool) error {
	current := llm.ConfigFromEnv()

	provider := strings.ToLower(strings.TrimSpace(p.Provider))
	if provider == "" {
		provider = "deepseek"
	}
	if _, ok := llm.GetPreset(provider); !ok && provider != "custom" {
		return fmt.Errorf("不支持的 LLM_PROVIDER: %s", provider)
	}

	baseURL := strings.TrimSpace(p.BaseURL)
	model := strings.TrimSpace(p.Model)
	if provider != "custom" {
		if preset, ok := llm.GetPreset(provider); ok {
			if baseURL == "" {
				baseURL = preset.BaseURL
			}
			if model == "" {
				model = preset.Model
			}
		}
	}
	if provider == "custom" {
		if baseURL == "" {
			return fmt.Errorf("自定义提供商须填写 Base URL")
		}
		if model == "" {
			return fmt.Errorf("自定义提供商须填写模型名称")
		}
	}

	updates := map[string]string{
		"LLM_PROVIDER": provider,
		"LLM_BASE_URL": baseURL,
		"LLM_MODEL":    model,
	}

	if forceAPIKey {
		updates["LLM_API_KEY"] = strings.TrimSpace(p.APIKey)
	} else if err := mergeSecret(updates, "LLM_API_KEY", p.APIKey, current.APIKey); err != nil {
		return err
	}

	if err := UpdateEnvKeys(llmEnvPath(), updates); err != nil {
		return err
	}
	if !forceAPIKey && strings.TrimSpace(p.APIKey) != "" {
		return SetProfilesGlobalAPIKey(p.APIKey)
	}
	return nil
}

var llmEnvFile = DefaultEnvPath

// SetLLMEnvFile 测试或自定义 .env 路径
func SetLLMEnvFile(path string) {
	llmEnvFile = path
}

func llmEnvPath() string {
	if llmEnvFile == "" {
		return DefaultEnvPath
	}
	return llmEnvFile
}
