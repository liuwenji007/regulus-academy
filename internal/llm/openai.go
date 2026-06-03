package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIConfig OpenAI 兼容 API 配置
type OpenAIConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

// OpenAIClient OpenAI 兼容 chat/completions 客户端
type OpenAIClient struct {
	provider   string
	display    string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAI 创建 OpenAI 兼容客户端
func NewOpenAI(cfg OpenAIConfig) *OpenAIClient {
	display := cfg.Provider
	if p, ok := GetPreset(cfg.Provider); ok && p.Name != "" {
		display = p.Name
	}
	return &OpenAIClient{
		provider: cfg.Provider,
		display:  display,
		apiKey:   cfg.APIKey,
		baseURL:  normalizeBaseURL(cfg.BaseURL),
		model:    cfg.Model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OpenAIClient) Configured() bool {
	return c.apiKey != "" || c.provider == "ollama"
}

func (c *OpenAIClient) Name() string {
	if c.display != "" {
		return c.display
	}
	return c.provider
}

func (c *OpenAIClient) Model() string {
	return c.model
}

type chatRequest struct {
	Model          string           `json:"model"`
	Messages       []Message        `json:"messages"`
	Temperature    float64          `json:"temperature,omitempty"`
	ResponseFormat *responseFormat  `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (string, error) {
	return c.ChatWithTemp(ctx, messages, 0.6)
}

func (c *OpenAIClient) ChatWithTemp(ctx context.Context, messages []Message, temp float64) (string, error) {
	return c.chatCompletion(ctx, messages, temp, false)
}

func (c *OpenAIClient) supportsJSONMode() bool {
	switch c.provider {
	case "deepseek", "openai", "openrouter":
		return true
	default:
		return false
	}
}

func (c *OpenAIClient) chatCompletion(ctx context.Context, messages []Message, temp float64, jsonMode bool) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}

	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temp,
	}
	if jsonMode && c.supportsJSONMode() {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 %s 失败: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s 返回错误 (HTTP %d): %s", c.Name(), resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("%s API 错误: %s", c.Name(), result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%s 返回空结果", c.Name())
	}
	return result.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) ChatJSON(ctx context.Context, messages []Message, temp float64, dest any) error {
	useJSONMode := c.supportsJSONMode()
	raw, err := c.chatCompletion(ctx, messages, temp, useJSONMode)
	if err != nil {
		return err
	}
	raw = extractJSON(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		retryMsg := Message{Role: "user", Content: "你上次输出不是合法 JSON，请只输出 JSON，不要 markdown 代码块。"}
		messages = append(messages, retryMsg)
		raw2, err2 := c.chatCompletion(ctx, messages, temp, useJSONMode)
		if err2 != nil {
			return fmt.Errorf("重试 LLM 请求失败: %w", err2)
		}
		raw2 = extractJSON(raw2)
		if err3 := json.Unmarshal([]byte(raw2), dest); err3 != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err3)
		}
		return nil
	}
	return nil
}

func (c *OpenAIClient) Ping(ctx context.Context) error {
	_, err := c.Chat(ctx, []Message{{Role: "user", Content: "ping"}})
	return err
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			start := 1
			end := len(lines) - 1
			if lines[end] == "```" {
				return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
			}
		}
	}
	return s
}

// NewClient 兼容旧接口：DeepSeek + 自定义 baseURL
func NewClient(apiKey, baseURL string) Provider {
	return NewFromConfig(OpenAIConfig{
		Provider: "deepseek",
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    "deepseek-chat",
	})
}
