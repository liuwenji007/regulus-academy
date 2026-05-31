package channel

import (
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

// PlatformUserAllowed 检查平台用户是否在 allowlist 中（空 allowlist 表示允许全部）
func PlatformUserAllowed(platform, platformUserID string) bool {
	platformUserID = strings.TrimSpace(platformUserID)
	if platformUserID == "" {
		return false
	}
	cfg := config.GatewayFromEnv()
	var allowed []string
	switch platform {
	case PlatformTelegram:
		allowed = cfg.Telegram.AllowedUsers
	case PlatformFeishu:
		allowed = cfg.Feishu.AllowedUsers
	case PlatformWeCom:
		allowed = cfg.WeCom.AllowedUsers
	default:
		return true
	}
	if len(allowed) == 0 {
		return true
	}
	for _, id := range allowed {
		if id == platformUserID {
			return true
		}
	}
	return false
}
