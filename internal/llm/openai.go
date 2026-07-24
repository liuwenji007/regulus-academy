package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/observability"
)

// OpenAIConfig OpenAI 兼容 API 配置
type OpenAIConfig struct {
	Provider    string
	APIKey      string
	BaseURL     string
	Model       string
	HTTPTimeout time.Duration // 0 表示使用 REGULUS_LLM_TIMEOUT_SEC（默认 240s）
	MaxTokens   int           // JSON 模式 max_tokens；0 表示 REGULUS_LLM_MAX_TOKENS（默认 8192）
}

// OpenAIClient OpenAI 兼容 chat/completions 客户端
type OpenAIClient struct {
	provider   string
	display    string
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// NewOpenAI 创建 OpenAI 兼容客户端
func NewOpenAI(cfg OpenAIConfig) *OpenAIClient {
	display := cfg.Provider
	if p, ok := GetPreset(cfg.Provider); ok && p.Name != "" {
		display = p.Name
	}
	httpTimeout := cfg.HTTPTimeout
	if httpTimeout <= 0 {
		httpTimeout = HTTPTimeoutFromEnv()
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = MaxTokensFromEnv()
	}
	return &OpenAIClient{
		provider:  cfg.Provider,
		display:   display,
		apiKey:    cfg.APIKey,
		baseURL:   normalizeBaseURL(cfg.BaseURL),
		model:     cfg.Model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: httpTimeout,
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
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
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

func (c *OpenAIClient) ChatStream(ctx context.Context, messages []Message, temp float64, onDelta func(string)) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}

	obsMsgs := make([]observability.ChatMessage, len(messages))
	for i, m := range messages {
		obsMsgs[i] = observability.ChatMessage{Role: m.Role, Content: m.Content}
	}

	start := time.Now()
	task := observability.GenerationFromContext(ctx)
	if task == "" {
		task = "llm.chat_stream"
	}

	out, err := observability.ObserveChatCompletion(ctx, c.provider, c.model, obsMsgs, temp, false,
		func(ctx context.Context) (string, error) {
			return c.doChatStream(ctx, messages, temp, onDelta)
		})
	log.Printf("llm.task=%s provider=%s model=%s stream=true duration_ms=%d err=%v",
		task, c.Name(), c.model, time.Since(start).Milliseconds(), err != nil)
	return out, err
}

type streamChatResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenAIClient) doChatStream(ctx context.Context, messages []Message, temp float64, onDelta func(string)) (string, error) {
	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temp,
		Stream:      true,
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
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 %s 失败: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%s 返回错误 (HTTP %d): %s", c.Name(), resp.StatusCode, string(respBody))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	// 单行 SSE data 可能较长
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if payload == "[DONE]" {
				break
			}
			continue
		}
		var chunk streamChatResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return "", fmt.Errorf("%s API 错误: %s", c.Name(), chunk.Error.Message)
		}
		if chunk.Usage != nil {
			u := TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
			if u.TotalTokens == 0 {
				u.TotalTokens = u.PromptTokens + u.CompletionTokens
			}
			reportUsage(ctx, u)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取流式响应失败: %w", err)
	}
	out := full.String()
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("%s 流式返回空内容", c.Name())
	}
	return out, nil
}

func (c *OpenAIClient) supportsJSONMode() bool {
	switch c.provider {
	case "deepseek", "openai", "openrouter":
		return true
	default:
		return false
	}
}

func (c *OpenAIClient) resolveMaxTokens(ctx context.Context, jsonMode bool) int {
	if !jsonMode || !c.supportsJSONMode() {
		return 0
	}
	if v, ok := jsonMaxTokensFromContext(ctx); ok {
		return v
	}
	return c.maxTokens
}

func (c *OpenAIClient) chatCompletion(ctx context.Context, messages []Message, temp float64, jsonMode bool) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}

	maxTokens := c.resolveMaxTokens(ctx, jsonMode)
	obsMsgs := make([]observability.ChatMessage, len(messages))
	for i, m := range messages {
		obsMsgs[i] = observability.ChatMessage{Role: m.Role, Content: m.Content}
	}

	start := time.Now()
	task := observability.GenerationFromContext(ctx)
	if task == "" {
		if jsonMode {
			task = "llm.chat_json"
		} else {
			task = "llm.chat"
		}
	}

	out, err := observability.ObserveChatCompletion(ctx, c.provider, c.model, obsMsgs, temp, jsonMode,
		func(ctx context.Context) (string, error) {
			out, err := c.doChatCompletion(ctx, messages, temp, jsonMode, maxTokens)
			if err == nil && strings.TrimSpace(out) == "" {
				log.Printf("%s 返回空内容，同轮重试", c.Name())
				out, err = c.doChatCompletion(ctx, messages, temp, jsonMode, maxTokens)
				if err == nil && strings.TrimSpace(out) == "" {
					return "", fmt.Errorf("%s 返回空内容", c.Name())
				}
			}
			return out, err
		})
	log.Printf("llm.task=%s provider=%s model=%s stream=false duration_ms=%d err=%v",
		task, c.Name(), c.model, time.Since(start).Milliseconds(), err != nil)
	return out, err
}

func (c *OpenAIClient) doChatCompletion(ctx context.Context, messages []Message, temp float64, jsonMode bool, maxTokens int) (string, error) {
	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temp,
	}
	if jsonMode && c.supportsJSONMode() {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
		if maxTokens > 0 {
			reqBody.MaxTokens = maxTokens
		}
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
	if result.Usage != nil {
		u := TokenUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
		if u.TotalTokens == 0 {
			u.TotalTokens = u.PromptTokens + u.CompletionTokens
		}
		reportUsage(ctx, u)
	}
	return result.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) ChatJSON(ctx context.Context, messages []Message, temp float64, dest any) error {
	useJSONMode := c.supportsJSONMode()
	maxTokens := c.resolveMaxTokens(ctx, useJSONMode)
	raw, err := c.chatCompletion(ctx, messages, temp, useJSONMode)
	if err != nil {
		return err
	}
	raw = extractJSON(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		retryCtx := ctx
		if isTruncatedJSONErr(err) {
			retryMax := bumpMaxTokens(maxTokens)
			if retryMax > 0 && retryMax != maxTokens {
				log.Printf("LLM JSON 可能被截断（max_tokens=%d），同轮重试并提高至 %d: %v", maxTokens, retryMax, err)
				retryCtx = WithJSONMaxTokens(ctx, retryMax)
			} else {
				log.Printf("LLM JSON 可能被截断（未设 max_tokens 上限），同轮重试并提示精简: %v", err)
			}
		} else {
			log.Printf("LLM JSON 解析失败，同轮重试: %v（提取后前缀 %q）", err, logJSONPrefix(raw))
		}
		messages = append(messages, Message{Role: "assistant", Content: raw})
		retryMsg := Message{Role: "user", Content: jsonRetryUserMessage(err)}
		messages = append(messages, retryMsg)
		raw2, err2 := c.chatCompletion(retryCtx, messages, temp, useJSONMode)
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

// ChatPromptJSON 通过普通对话请求 JSON（不启用 response_format json_object）。
// 对话类结构化输出更稳定，可避免 DeepSeek 等在 json_object 模式下频繁返回空内容。
func ChatPromptJSON(ctx context.Context, provider Provider, messages []Message, temp float64, dest any) error {
	if provider == nil {
		return fmt.Errorf("未配置 LLM")
	}
	return parseJSONFromChat(ctx, provider, messages, temp, dest)
}

func parseJSONFromChat(ctx context.Context, provider Provider, messages []Message, temp float64, dest any) error {
	raw, err := provider.ChatWithTemp(ctx, messages, temp)
	if err != nil {
		return err
	}
	raw = extractJSON(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		if isTruncatedJSONErr(err) {
			log.Printf("LLM prompt JSON 可能被截断，同轮重试: %v", err)
		} else {
			log.Printf("LLM prompt JSON 解析失败，同轮重试: %v（提取后前缀 %q）", err, logJSONPrefix(raw))
		}
		messages = append(messages, Message{Role: "assistant", Content: raw})
		retryMsg := Message{Role: "user", Content: jsonRetryUserMessage(err)}
		messages = append(messages, retryMsg)
		raw2, err2 := provider.ChatWithTemp(ctx, messages, temp)
		if err2 != nil {
			return fmt.Errorf("重试 LLM 请求失败: %w", err2)
		}
		raw2 = extractJSON(raw2)
		if err3 := json.Unmarshal([]byte(raw2), dest); err3 != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err3)
		}
	}
	return nil
}

func (c *OpenAIClient) Ping(ctx context.Context) error {
	_, err := c.Chat(ctx, []Message{{Role: "user", Content: "ping"}})
	return err
}

func logJSONPrefix(raw string) string {
	const n = 120
	if len(raw) <= n {
		return raw
	}
	return raw[:n] + "…"
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
