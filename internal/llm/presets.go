package llm

// Preset 内置模型提供商预设
type Preset struct {
	Name    string
	BaseURL string
	Model   string
}

var presets = map[string]Preset{
	"deepseek": {
		Name:    "DeepSeek",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat",
	},
	"openai": {
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com",
		Model:   "gpt-4o-mini",
	},
	"openrouter": {
		Name:    "OpenRouter",
		BaseURL: "https://openrouter.ai/api",
		Model:   "deepseek/deepseek-chat",
	},
	"ollama": {
		Name:    "Ollama",
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
	},
	"custom": {
		Name: "Custom",
	},
}

// ListPresets 返回可用预设 id 列表
func ListPresets() []string {
	ids := make([]string, 0, len(presets))
	for id := range presets {
		ids = append(ids, id)
	}
	return ids
}

// GetPreset 读取预设，不存在时 ok=false
func GetPreset(id string) (Preset, bool) {
	p, ok := presets[id]
	return p, ok
}
