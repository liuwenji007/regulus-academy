package observability

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ChatMessage LLM 消息摘要（用于 span 属性）
type ChatMessage struct {
	Role    string
	Content string
}

// ObserveChatCompletion 包装一次 chat/completions 调用并记录 OTel generation span
func ObserveChatCompletion(
	ctx context.Context,
	providerName, model string,
	messages []ChatMessage,
	temperature float64,
	jsonMode bool,
	fn func(context.Context) (string, error),
) (string, error) {
	if !Enabled() {
		return fn(ctx)
	}

	genName := GenerationFromContext(ctx)
	if genName == "" {
		genName = "llm.chat"
	}

	ctx, span := globalTracer.Start(ctx, genName, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	applyTraceMetaToSpan(span, ctx)

	attrs := []attribute.KeyValue{
		attribute.String("langfuse.observation.type", "generation"),
		attribute.String("gen_ai.system", providerName),
		attribute.String("gen_ai.request.model", model),
		attribute.Float64("gen_ai.request.temperature", temperature),
		attribute.Bool("gen_ai.request.json_mode", jsonMode),
		attribute.Int("gen_ai.request.message_count", len(messages)),
		attribute.String("langfuse.environment", globalCfg.Environment),
	}
	if LogContent() {
		attrs = append(attrs,
			attribute.String("gen_ai.prompt", formatMessages(messages)),
		)
	}
	span.SetAttributes(attrs...)

	start := time.Now()
	out, err := fn(ctx)
	span.SetAttributes(attribute.Int64("gen_ai.response.duration_ms", time.Since(start).Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	if LogContent() {
		span.SetAttributes(attribute.String("gen_ai.completion", truncate(out, 8000)))
	}
	span.SetStatus(codes.Ok, "")
	return out, nil
}

func formatMessages(msgs []ChatMessage) string {
	var b strings.Builder
	for i, m := range msgs {
		if i > 0 {
			b.WriteByte('\n')
		}
		role := m.Role
		if role == "" {
			role = "unknown"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(truncate(strings.TrimSpace(m.Content), 2000))
	}
	return b.String()
}
