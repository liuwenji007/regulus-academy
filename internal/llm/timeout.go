package llm

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPTimeoutSec       = 240
	defaultDomainBuildTimeoutSec = 360
)

// HTTPTimeoutFromEnv 单次 LLM HTTP 请求超时（秒）；REGULUS_LLM_TIMEOUT_SEC，默认 240。
func HTTPTimeoutFromEnv() time.Duration {
	return durationFromEnv("REGULUS_LLM_TIMEOUT_SEC", defaultHTTPTimeoutSec)
}

// DomainBuildTimeoutFromEnv 建树/regenerate 整请求超时；REGULUS_DOMAIN_BUILD_TIMEOUT_SEC，默认 360。
func DomainBuildTimeoutFromEnv() time.Duration {
	return durationFromEnv("REGULUS_DOMAIN_BUILD_TIMEOUT_SEC", defaultDomainBuildTimeoutSec)
}

func durationFromEnv(key string, defaultSec int) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return time.Duration(defaultSec) * time.Second
	}
	sec, err := strconv.Atoi(v)
	if err != nil || sec < 30 {
		return time.Duration(defaultSec) * time.Second
	}
	return time.Duration(sec) * time.Second
}

// IsTimeoutErr 判断是否为请求/上下文超时。
func IsTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout exceeded") ||
		strings.Contains(msg, "timeout awaiting response headers")
}
