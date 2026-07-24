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
	// ChatStream 流式对话；onDelta 可为 nil。返回完整文本便于落库。
	ChatStream(ctx context.Context, messages []Message, temp float64, onDelta func(string)) (string, error)
	ChatJSON(ctx context.Context, messages []Message, temp float64, dest any) error
	Ping(ctx context.Context) error
}

// StreamViaChat 用非流式 ChatWithTemp 模拟 ChatStream（测试 mock / 无流式能力时回退）。
func StreamViaChat(
	ctx context.Context,
	chat func(context.Context, []Message, float64) (string, error),
	messages []Message,
	temp float64,
	onDelta func(string),
) (string, error) {
	out, err := chat(ctx, messages, temp)
	if err != nil {
		return "", err
	}
	if onDelta != nil && out != "" {
		onDelta(out)
	}
	return out, nil
}
