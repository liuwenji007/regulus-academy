package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

// LLMProfile 用户保存的一条模型配置（可自定义显示名称）
type LLMProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey,omitempty"`
}

// LLMProfilesState 持久化的模型列表
type LLMProfilesState struct {
	ActiveID     string       `json:"activeId"`
	GlobalAPIKey string       `json:"globalApiKey,omitempty"`
	Profiles     []LLMProfile `json:"profiles"`
}

var llmProfilesFile = ""

// SetLLMProfilesFile 测试或自定义路径
func SetLLMProfilesFile(path string) {
	llmProfilesFile = path
}

func llmProfilesPath() string {
	if llmProfilesFile != "" {
		return llmProfilesFile
	}
	db := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	if db == "" {
		db = "./data/regulus.db"
	}
	return filepath.Join(filepath.Dir(db), "llm-profiles.json")
}

// LoadLLMProfiles 读取模型列表；不存在时从当前 .env 迁移一条默认配置
func LoadLLMProfiles() (LLMProfilesState, error) {
	path := llmProfilesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return migrateProfilesFromEnv(), nil
		}
		return LLMProfilesState{}, err
	}
	var state LLMProfilesState
	if err := json.Unmarshal(data, &state); err != nil {
		return LLMProfilesState{}, fmt.Errorf("解析模型配置失败: %w", err)
	}
	if len(state.Profiles) == 0 {
		return migrateProfilesFromEnv(), nil
	}
	if state.ActiveID == "" {
		state.ActiveID = state.Profiles[0].ID
	}
	backfillGlobalAPIKey(&state)
	return state, nil
}

// backfillGlobalAPIKey 兼容旧版 llm-profiles.json（无 globalApiKey 字段）
func backfillGlobalAPIKey(state *LLMProfilesState) {
	if strings.TrimSpace(state.GlobalAPIKey) != "" {
		return
	}
	state.GlobalAPIKey = strings.TrimSpace(llm.ConfigFromEnv().APIKey)
}

// MigrateProfilesFromEnv 从当前 .env 生成一条默认模型配置
func MigrateProfilesFromEnv() LLMProfilesState {
	return migrateProfilesFromEnv()
}

func migrateProfilesFromEnv() LLMProfilesState {
	cfg := llm.ConfigFromEnv()
	name := cfg.Model
	if p, ok := llm.GetPreset(cfg.Provider); ok && p.Name != "" {
		name = p.Name + " · " + cfg.Model
	}
	id := "default"
	return LLMProfilesState{
		ActiveID:     id,
		GlobalAPIKey: cfg.APIKey,
		Profiles: []LLMProfile{{
			ID:       id,
			Name:     name,
			Provider: cfg.Provider,
			BaseURL:  cfg.BaseURL,
			Model:    cfg.Model,
		}},
	}
}

// MergeProfileAPIKeysFromExisting 保留已有条目中的专属 API Key（Web 保存时密钥字段留空不传）
func MergeProfileAPIKeysFromExisting(existing, incoming LLMProfilesState) LLMProfilesState {
	if len(existing.Profiles) == 0 {
		return incoming
	}
	byID := make(map[string]string, len(existing.Profiles))
	for _, p := range existing.Profiles {
		if k := strings.TrimSpace(p.APIKey); k != "" {
			byID[p.ID] = k
		}
	}
	for i := range incoming.Profiles {
		if strings.TrimSpace(incoming.Profiles[i].APIKey) != "" {
			continue
		}
		if k, ok := byID[incoming.Profiles[i].ID]; ok {
			incoming.Profiles[i].APIKey = k
		}
	}
	return incoming
}

// SaveLLMProfiles 写入文件
func SaveLLMProfiles(state LLMProfilesState) error {
	if err := validateProfilesState(&state); err != nil {
		return err
	}
	path := llmProfilesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func validateProfilesState(state *LLMProfilesState) error {
	if len(state.Profiles) == 0 {
		return fmt.Errorf("至少保留一条模型配置")
	}
	activeOK := false
	seen := make(map[string]bool)
	for i := range state.Profiles {
		p := &state.Profiles[i]
		p.ID = strings.TrimSpace(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.Provider = strings.ToLower(strings.TrimSpace(p.Provider))
		p.BaseURL = strings.TrimSpace(p.BaseURL)
		p.Model = strings.TrimSpace(p.Model)
		p.APIKey = strings.TrimSpace(p.APIKey)

		if p.ID == "" {
			p.ID = NewProfileID()
		}
		if p.Name == "" {
			return fmt.Errorf("第 %d 条模型须填写显示名称", i+1)
		}
		if _, ok := llm.GetPreset(p.Provider); !ok && p.Provider != "custom" {
			return fmt.Errorf("不支持的提供商: %s", p.Provider)
		}
		if err := normalizeProfileEndpoints(p); err != nil {
			return fmt.Errorf("%s: %w", p.Name, err)
		}
		if seen[p.ID] {
			return fmt.Errorf("重复的模型 ID: %s", p.ID)
		}
		seen[p.ID] = true
		if p.ID == state.ActiveID {
			activeOK = true
		}
	}
	if !activeOK && len(state.Profiles) > 0 {
		state.ActiveID = state.Profiles[0].ID
	}
	return nil
}

func normalizeProfileEndpoints(p *LLMProfile) error {
	if p.Provider != "custom" {
		if preset, ok := llm.GetPreset(p.Provider); ok {
			if p.BaseURL == "" {
				p.BaseURL = preset.BaseURL
			}
			if p.Model == "" {
				p.Model = preset.Model
			}
		}
		return nil
	}
	if p.BaseURL == "" {
		return fmt.Errorf("自定义接口须填写 Base URL")
	}
	if p.Model == "" {
		return fmt.Errorf("须填写模型 ID")
	}
	return nil
}

// ApplyActiveLLMProfile 将当前选中模型写入 .env 并更新环境变量
func ApplyActiveLLMProfile(state LLMProfilesState) error {
	var active *LLMProfile
	for i := range state.Profiles {
		if state.Profiles[i].ID == state.ActiveID {
			active = &state.Profiles[i]
			break
		}
	}
	if active == nil {
		return fmt.Errorf("未找到当前模型配置")
	}
	apiKey := strings.TrimSpace(active.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(state.GlobalAPIKey)
	}
	payload := LLMSettingsPayload{
		Provider: active.Provider,
		BaseURL:  active.BaseURL,
		Model:    active.Model,
		APIKey:   apiKey,
	}
	return applyLLMSettings(payload, true)
}

// SetProfilesGlobalAPIKey 更新持久化的全局 Key（单表单保存或安装时写入 .env 后同步）
func SetProfilesGlobalAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	state, err := LoadLLMProfiles()
	if err != nil {
		return err
	}
	state.GlobalAPIKey = key
	return SaveLLMProfiles(state)
}

// LLMProfileView Web 列表项（密钥脱敏）
type LLMProfileView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	BaseURL    string `json:"baseUrl,omitempty"`
	Model      string `json:"model"`
	APIKeySet  bool   `json:"apiKeySet"`
}

// ProfilesViewFromState 构建 API 视图
func ProfilesViewFromState(state LLMProfilesState) []LLMProfileView {
	globalKey := strings.TrimSpace(llm.ConfigFromEnv().APIKey)
	out := make([]LLMProfileView, 0, len(state.Profiles))
	for _, p := range state.Profiles {
		keySet := strings.TrimSpace(p.APIKey) != "" || p.Provider == "ollama" || globalKey != ""
		out = append(out, LLMProfileView{
			ID:        p.ID,
			Name:      p.Name,
			Provider:  p.Provider,
			BaseURL:   p.BaseURL,
			Model:     p.Model,
			APIKeySet: keySet,
		})
	}
	return out
}

// NewProfileID 生成新模型 ID
func NewProfileID() string {
	return "m-" + time.Now().UTC().Format("20060102150405")
}
