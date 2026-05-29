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

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client DeepSeek API 客户端
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 DeepSeek 客户端
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Configured 是否已配置 API Key
func (c *Client) Configured() bool {
	return c.apiKey != ""
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat 调用 DeepSeek 对话补全
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	return c.ChatWithTemp(ctx, messages, 0.6)
}

// ChatWithTemp 指定 temperature
func (c *Client) ChatWithTemp(ctx context.Context, messages []Message, temp float64) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("未配置 DEEPSEEK_API_KEY")
	}

	body, err := json.Marshal(chatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Temperature: temp,
	})
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 DeepSeek 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek 返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("DeepSeek API 错误: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek 返回空结果")
	}
	return result.Choices[0].Message.Content, nil
}

// ChatJSON 要求模型返回 JSON 并解析到 dest
func (c *Client) ChatJSON(ctx context.Context, messages []Message, temp float64, dest any) error {
	raw, err := c.ChatWithTemp(ctx, messages, temp)
	if err != nil {
		return err
	}
	raw = extractJSON(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		// 重试一次
		retryMsg := Message{Role: "user", Content: "你上次输出不是合法 JSON，请只输出 JSON，不要 markdown 代码块。"}
		messages = append(messages, retryMsg)
		raw2, err2 := c.ChatWithTemp(ctx, messages, temp)
		if err2 != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err)
		}
		raw2 = extractJSON(raw2)
		if err3 := json.Unmarshal([]byte(raw2), dest); err3 != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err3)
		}
		return nil
	}
	return nil
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

// Ping 发送最小请求验证 API Key
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Chat(ctx, []Message{{Role: "user", Content: "ping"}})
	return err
}
