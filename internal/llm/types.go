package llm

import "context"

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider 大模型调用接口（OpenAI 兼容协议）
type Provider interface {
	Configured() bool
	Name() string
	Model() string
	Chat(ctx context.Context, messages []Message) (string, error)
	ChatWithTemp(ctx context.Context, messages []Message, temp float64) (string, error)
	ChatJSON(ctx context.Context, messages []Message, temp float64, dest any) error
	Ping(ctx context.Context) error
}
