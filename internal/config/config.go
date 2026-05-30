package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

// Config 应用配置
type Config struct {
	Port         string
	DatabasePath string
	LLM          llm.OpenAIConfig
	Gateway      GatewayConfig
}

// Load 从环境变量加载配置，并尝试读取 .env 文件
func Load() *Config {
	loadEnvFile(".env")

	port := getEnv("PORT", "8080")
	dbPath := getEnv("DATABASE_PATH", "./data/regulus.db")
	llmCfg := llm.ConfigFromEnv()

	return &Config{
		Port:         port,
		DatabasePath: dbPath,
		LLM:          llmCfg,
		Gateway:      GatewayFromEnv(),
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Addr 返回监听地址
func (c *Config) Addr() string {
	if _, err := strconv.Atoi(c.Port); err != nil {
		return ":8080"
	}
	return ":" + c.Port
}
