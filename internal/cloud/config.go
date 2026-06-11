package cloud

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const DeploymentCloud = "cloud"

// Config Cloud Demo 运行时配置
type Config struct {
	Deployment          string
	QuotaDailyMessages  int
	MaxBuildJobsGlobal  int
	RateLimitPerIP      int
	GithubURL           string
	DocsURL             string
	DemoURL             string
	EncryptionKey       string
	AdminTokenRequired  bool
}

// LoadConfig 从环境变量加载；非 cloud 模式返回 Enabled=false 的零值配置
func LoadConfig() Config {
	deployment := strings.TrimSpace(os.Getenv("REGULUS_DEPLOYMENT"))
	if deployment != DeploymentCloud {
		return Config{Deployment: deployment}
	}

	cfg := Config{
		Deployment:         DeploymentCloud,
		QuotaDailyMessages: envIntDefault("REGULUS_CLOUD_QUOTA_DAILY_MESSAGES", 20),
		MaxBuildJobsGlobal: envIntDefault("REGULUS_CLOUD_MAX_BUILD_JOBS_GLOBAL", 3),
		RateLimitPerIP:     envIntDefault("REGULUS_CLOUD_RATE_LIMIT_PER_IP", 60),
		GithubURL:          strings.TrimSpace(os.Getenv("REGULUS_CLOUD_GITHUB_URL")),
		DocsURL:            strings.TrimSpace(os.Getenv("REGULUS_CLOUD_DOCS_URL")),
		DemoURL:            strings.TrimSpace(os.Getenv("REGULUS_CLOUD_DEMO_URL")),
		EncryptionKey:      strings.TrimSpace(os.Getenv("REGULUS_CLOUD_ENCRYPTION_KEY")),
		AdminTokenRequired: true,
	}

	if cfg.GithubURL == "" {
		cfg.GithubURL = "https://github.com/liuwenji007/regulus-academy"
	}

	adminToken := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	if adminToken == "" {
		log.Fatal("[cloud] REGULUS_DEPLOYMENT=cloud 时必须设置 ADMIN_TOKEN")
	}
	if cfg.EncryptionKey == "" {
		log.Fatal("[cloud] REGULUS_DEPLOYMENT=cloud 时必须设置 REGULUS_CLOUD_ENCRYPTION_KEY")
	}

	if strings.TrimSpace(os.Getenv("GATEWAY_ENABLED")) == "true" {
		log.Println("[cloud] 警告: 公网 Demo 建议保持 GATEWAY_ENABLED=false")
	}

	return cfg
}

func (c Config) Enabled() bool {
	return c.Deployment == DeploymentCloud
}

func envIntDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
