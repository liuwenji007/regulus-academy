package config

import (
	"os"
	"strings"
)

// GatewayConfig IM Gateway 配置
type GatewayConfig struct {
	Enabled bool

	Telegram TelegramConfig
	DingTalk DingTalkConfig
	Feishu   FeishuConfig
	WeCom    WeComConfig
}

// TelegramConfig Telegram Bot
type TelegramConfig struct {
	Enabled      bool
	BotToken     string
	AllowedUsers []string
}

// DingTalkConfig 钉钉 Stream 机器人
type DingTalkConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
}

// FeishuConfig 飞书机器人
type FeishuConfig struct {
	Enabled   bool
	AppID     string
	AppSecret string
	// Mode: websocket（默认，长连接，无需公网）| webhook（HTTP 回调，需公网 HTTPS）
	Mode string
}

// WeComConfig 企业微信应用回调
type WeComConfig struct {
	Enabled          bool
	CorpID           string
	AgentID          string
	Secret           string
	Token            string
	EncodingAESKey   string
	AllowedUsers     []string
}

// GatewayFromEnv 从环境变量加载 Gateway 配置
func GatewayFromEnv() GatewayConfig {
	enabled := envBool("GATEWAY_ENABLED", false)
	return GatewayConfig{
		Enabled: enabled,
		Telegram: TelegramConfig{
			Enabled:      enabled && envBool("TELEGRAM_ENABLED", true),
			BotToken:     os.Getenv("TELEGRAM_BOT_TOKEN"),
			AllowedUsers: splitCSV(os.Getenv("TELEGRAM_ALLOWED_USERS")),
		},
		DingTalk: DingTalkConfig{
			Enabled:      enabled && envBool("DINGTALK_ENABLED", true),
			ClientID:     os.Getenv("DINGTALK_CLIENT_ID"),
			ClientSecret: os.Getenv("DINGTALK_CLIENT_SECRET"),
		},
		Feishu: FeishuConfig{
			Enabled:   enabled && envBool("FEISHU_ENABLED", true),
			AppID:     os.Getenv("FEISHU_APP_ID"),
			AppSecret: os.Getenv("FEISHU_APP_SECRET"),
			Mode:      feishuModeFromEnv(),
		},
		WeCom: WeComConfig{
			Enabled:        enabled && envBool("WECOM_ENABLED", false),
			CorpID:         os.Getenv("WECOM_CORP_ID"),
			AgentID:        os.Getenv("WECOM_AGENT_ID"),
			Secret:         os.Getenv("WECOM_SECRET"),
			Token:          os.Getenv("WECOM_TOKEN"),
			EncodingAESKey: os.Getenv("WECOM_ENCODING_AES_KEY"),
			AllowedUsers:   splitCSV(os.Getenv("WECOM_ALLOWED_USERS")),
		},
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes"
}

func feishuModeFromEnv() string {
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("FEISHU_MODE")))
	if mode == "webhook" {
		return "webhook"
	}
	return "websocket"
}
