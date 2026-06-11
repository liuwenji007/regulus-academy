package llm

import "context"

type providerOverrideKey struct{}

// WithProvider 在 context 中注入本次请求使用的 LLM（Cloud BYOK 等）
func WithProvider(ctx context.Context, p Provider) context.Context {
	if p == nil {
		return ctx
	}
	return context.WithValue(ctx, providerOverrideKey{}, p)
}

// ProviderFromContext 优先返回 context 中的 Provider，否则 fallback
func ProviderFromContext(ctx context.Context, fallback Provider) Provider {
	if ctx != nil {
		if p, ok := ctx.Value(providerOverrideKey{}).(Provider); ok && p != nil {
			return p
		}
	}
	return fallback
}
