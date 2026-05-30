package channel

import "context"

// Adapter 平台消息适配器
type Adapter interface {
	Name() string
	Start(ctx context.Context, onMessage func(MessageEvent)) error
	SendText(ctx context.Context, target ReplyTarget, text string) error
}
