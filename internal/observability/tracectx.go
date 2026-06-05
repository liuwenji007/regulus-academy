package observability

import (
	"context"
	"unicode/utf8"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey int

const (
	keyTraceMeta ctxKey = iota
	keyGeneration
)

// TraceMeta 用户可见操作的 trace 元数据（会映射到 Langfuse trace / span 属性）
type TraceMeta struct {
	Name      string
	UserID    string
	SessionID string
	DomainID  string
	NodeKey   string
	Phase     string
	Channel   string
	Input     string // 可选，截断后的业务输入（非 LLM messages）
}

// WithTrace 在 context 中保存 trace 元数据（由 Trace 调用方设置）
func WithTrace(ctx context.Context, meta TraceMeta) context.Context {
	return context.WithValue(ctx, keyTraceMeta, meta)
}

// TraceMetaFromContext 读取 trace 元数据
func TraceMetaFromContext(ctx context.Context) (TraceMeta, bool) {
	v := ctx.Value(keyTraceMeta)
	if v == nil {
		return TraceMeta{}, false
	}
	meta, ok := v.(TraceMeta)
	return meta, ok
}

// WithGeneration 为下一次 LLM 调用设置 generation 名称（如 coach.grade）
func WithGeneration(ctx context.Context, name string) context.Context {
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, keyGeneration, name)
}

// GenerationFromContext 读取 generation 名称
func GenerationFromContext(ctx context.Context) string {
	v := ctx.Value(keyGeneration)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Trace 开始一个 trace/span（span 名为 meta.Name），返回带元数据的 context 与 end 回调
func Trace(ctx context.Context, meta TraceMeta) (context.Context, func()) {
	if !Enabled() {
		return WithTrace(ctx, meta), func() {}
	}
	ctx = WithTrace(ctx, meta)
	ctx, span := globalTracer.Start(ctx, meta.Name, trace.WithSpanKind(trace.SpanKindServer))
	setTraceAttributes(span, meta)
	return ctx, func() { span.End() }
}

// StartChildSpan 在已有 trace 下启动子 span，不覆盖 context 中的 TraceMeta（保留 UserID 等父级字段）。
func StartChildSpan(ctx context.Context, name string, extra TraceMeta) (context.Context, func()) {
	meta, ok := TraceMetaFromContext(ctx)
	if !ok {
		meta = extra
	} else {
		if extra.Input != "" {
			meta.Input = extra.Input
		}
		if extra.Phase != "" {
			meta.Phase = extra.Phase
		}
	}
	ctx = WithTrace(ctx, meta)
	if !Enabled() {
		return ctx, func() {}
	}
	ctx, span := globalTracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))
	setTraceAttributes(span, meta)
	return ctx, func() { span.End() }
}

func setTraceAttributes(span trace.Span, meta TraceMeta) {
	if !span.IsRecording() {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("langfuse.trace.name", meta.Name),
		attribute.String("langfuse.environment", globalCfg.Environment),
	}
	if meta.UserID != "" {
		attrs = append(attrs, attribute.String("langfuse.user.id", meta.UserID))
	}
	if meta.SessionID != "" {
		attrs = append(attrs, attribute.String("langfuse.session.id", meta.SessionID))
	}
	if meta.DomainID != "" {
		attrs = append(attrs, attribute.String("regulus.domain_id", meta.DomainID))
	}
	if meta.NodeKey != "" {
		attrs = append(attrs, attribute.String("regulus.node_key", meta.NodeKey))
	}
	if meta.Phase != "" {
		attrs = append(attrs, attribute.String("regulus.phase", meta.Phase))
	}
	if meta.Channel != "" {
		attrs = append(attrs, attribute.String("regulus.channel", meta.Channel))
	}
	if meta.Input != "" {
		attrs = append(attrs, attribute.String("langfuse.trace.input", truncate(meta.Input, 500)))
	}
	span.SetAttributes(attrs...)
}

func applyTraceMetaToSpan(span trace.Span, ctx context.Context) {
	meta, ok := TraceMetaFromContext(ctx)
	if !ok {
		return
	}
	setTraceAttributes(span, meta)
}

func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "…"
}
