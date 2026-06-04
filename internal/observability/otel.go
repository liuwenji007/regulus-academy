package observability

import (
	"context"
	"encoding/base64"
	"log"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const tracerName = "github.com/regulus-academy/regulus-academy"

var (
	globalCfg    Config
	globalTracer = otel.Tracer(tracerName)
)

// Enabled 是否已启用 Langfuse OTLP 导出
func Enabled() bool {
	return globalCfg.Enabled
}

// LogContent 是否在 generation span 中记录 prompt/completion 正文
func LogContent() bool {
	return globalCfg.LogContent
}

// Init 在 LANGFUSE_ENABLED=true 时注册 OTLP TracerProvider；返回 Shutdown（noop 时为空操作）
func Init(cfg Config) func(context.Context) error {
	globalCfg = cfg
	if !cfg.Enabled {
		return func(context.Context) error { return nil }
	}
	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		log.Println("[langfuse] LANGFUSE_ENABLED=true 但缺少 PUBLIC_KEY/SECRET_KEY，追踪已禁用")
		globalCfg.Enabled = false
		return func(context.Context) error { return nil }
	}
	endpoint := cfg.OTLPEndpoint()
	if endpoint == "" {
		log.Println("[langfuse] LANGFUSE_ENABLED=true 但 LANGFUSE_BASE_URL 为空，追踪已禁用")
		globalCfg.Enabled = false
		return func(context.Context) error { return nil }
	}

	auth := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))
	otelOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":                "Basic " + auth,
			"x-langfuse-ingestion-version": "4",
		}),
	}
	if strings.HasPrefix(strings.ToLower(endpoint), "http://") {
		otelOpts = append(otelOpts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(context.Background(), otelOpts...)
	if err != nil {
		log.Printf("[langfuse] 创建 OTLP exporter 失败: %v", err)
		globalCfg.Enabled = false
		return func(context.Context) error { return nil }
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			attribute.String("service.name", "regulus-academy"),
		),
	)
	if err != nil {
		log.Printf("[langfuse] 创建 resource 失败: %v", err)
		globalCfg.Enabled = false
		return func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	log.Printf("[langfuse] OTLP → %s (environment=%s)", cfg.BaseURL, cfg.Environment)

	return func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}
}
