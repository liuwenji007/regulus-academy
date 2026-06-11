package llm

import "context"

// TokenUsage OpenAI 兼容 usage 字段
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type usageReporterKey struct{}

// WithUsageReporter 在 context 中注册 LLM 调用完成后的 usage 回调
func WithUsageReporter(ctx context.Context, fn func(TokenUsage)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, usageReporterKey{}, fn)
}

func reportUsage(ctx context.Context, u TokenUsage) {
	if ctx == nil || u.TotalTokens == 0 && u.PromptTokens == 0 && u.CompletionTokens == 0 {
		return
	}
	fn, ok := ctx.Value(usageReporterKey{}).(func(TokenUsage))
	if !ok || fn == nil {
		return
	}
	fn(u)
}
