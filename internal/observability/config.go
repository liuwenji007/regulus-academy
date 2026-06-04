package observability

import (
	"os"
	"strconv"
	"strings"
)

// Config Langfuse / OTLP 开发期可观测性配置（仅 LANGFUSE_ENABLED 控制是否初始化）
type Config struct {
	Enabled     bool
	PublicKey   string
	SecretKey   string
	BaseURL     string
	Environment string
	LogContent  bool
}

// LoadConfigFromEnv 从环境变量加载 Langfuse 配置
func LoadConfigFromEnv() Config {
	return Config{
		Enabled:     envBool("LANGFUSE_ENABLED", false),
		PublicKey:   strings.TrimSpace(os.Getenv("LANGFUSE_PUBLIC_KEY")),
		SecretKey:   strings.TrimSpace(os.Getenv("LANGFUSE_SECRET_KEY")),
		BaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("LANGFUSE_BASE_URL")), "/"),
		Environment: envString("LANGFUSE_ENVIRONMENT", "development"),
		LogContent:  envBool("LANGFUSE_LOG_CONTENT", true),
	}
}

// OTLPEndpoint Langfuse traces 导出地址（Go otlptracehttp 需完整 path，不会自动拼 /v1/traces）
func (c Config) OTLPEndpoint() string {
	if c.BaseURL == "" {
		return ""
	}
	return c.BaseURL + "/api/public/otel/v1/traces"
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
