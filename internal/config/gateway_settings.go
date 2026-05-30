package config

import (
	"fmt"
	"os"
	"strings"
)

// GatewaySettingsPayload Web 端可编辑的 Gateway 配置
type GatewaySettingsPayload struct {
	Enabled   bool   `json:"enabled"`
	PublicURL string `json:"publicUrl"`

	TelegramEnabled      bool   `json:"telegramEnabled"`
	TelegramBotToken     string `json:"telegramBotToken,omitempty"`
	TelegramAllowedUsers string `json:"telegramAllowedUsers"`

	DingTalkEnabled      bool   `json:"dingtalkEnabled"`
	DingTalkClientID     string `json:"dingtalkClientId"`
	DingTalkClientSecret string `json:"dingtalkClientSecret,omitempty"`

	FeishuEnabled   bool   `json:"feishuEnabled"`
	FeishuMode      string `json:"feishuMode"`
	FeishuAppID     string `json:"feishuAppId"`
	FeishuAppSecret string `json:"feishuAppSecret,omitempty"`

	WeComEnabled        bool   `json:"wecomEnabled"`
	WeComCorpID         string `json:"wecomCorpId"`
	WeComAgentID        string `json:"wecomAgentId"`
	WeComSecret         string `json:"wecomSecret,omitempty"`
	WeComToken          string `json:"wecomToken,omitempty"`
	WeComEncodingAESKey string `json:"wecomEncodingAesKey,omitempty"`
	WeComAllowedUsers   string `json:"wecomAllowedUsers"`
}

// GatewaySettingsView GET 返回的可编辑视图（密钥脱敏）
type GatewaySettingsView struct {
	Enabled   bool   `json:"enabled"`
	PublicURL string `json:"publicUrl"`

	TelegramEnabled      bool   `json:"telegramEnabled"`
	TelegramBotTokenSet  bool   `json:"telegramBotTokenSet"`
	TelegramAllowedUsers string `json:"telegramAllowedUsers"`

	DingTalkEnabled          bool   `json:"dingtalkEnabled"`
	DingTalkClientID         string `json:"dingtalkClientId"`
	DingTalkClientSecretSet  bool   `json:"dingtalkClientSecretSet"`

	FeishuEnabled       bool   `json:"feishuEnabled"`
	FeishuMode          string `json:"feishuMode"`
	FeishuAppID         string `json:"feishuAppId"`
	FeishuAppSecretSet  bool   `json:"feishuAppSecretSet"`

	WeComEnabled             bool   `json:"wecomEnabled"`
	WeComCorpID              string `json:"wecomCorpId"`
	WeComAgentID             string `json:"wecomAgentId"`
	WeComSecretSet           bool   `json:"wecomSecretSet"`
	WeComTokenSet            bool   `json:"wecomTokenSet"`
	WeComEncodingAESKeySet   bool   `json:"wecomEncodingAesKeySet"`
	WeComAllowedUsers        string `json:"wecomAllowedUsers"`
}

// GatewaySettingsViewFromEnv 从当前环境变量构建可编辑视图
func GatewaySettingsViewFromEnv() GatewaySettingsView {
	cfg := GatewayFromEnv()
	return GatewaySettingsView{
		Enabled:   cfg.Enabled,
		PublicURL: os.Getenv("GATEWAY_PUBLIC_URL"),

		TelegramEnabled:      envBool("TELEGRAM_ENABLED", true),
		TelegramBotTokenSet:  strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) != "",
		TelegramAllowedUsers: strings.TrimSpace(os.Getenv("TELEGRAM_ALLOWED_USERS")),

		DingTalkEnabled:         envBool("DINGTALK_ENABLED", true),
		DingTalkClientID:        strings.TrimSpace(os.Getenv("DINGTALK_CLIENT_ID")),
		DingTalkClientSecretSet: strings.TrimSpace(os.Getenv("DINGTALK_CLIENT_SECRET")) != "",

		FeishuEnabled:      envBool("FEISHU_ENABLED", true),
		FeishuMode:         feishuModeFromEnv(),
		FeishuAppID:        strings.TrimSpace(os.Getenv("FEISHU_APP_ID")),
		FeishuAppSecretSet: strings.TrimSpace(os.Getenv("FEISHU_APP_SECRET")) != "",

		WeComEnabled:           envBool("WECOM_ENABLED", false),
		WeComCorpID:            strings.TrimSpace(os.Getenv("WECOM_CORP_ID")),
		WeComAgentID:           strings.TrimSpace(os.Getenv("WECOM_AGENT_ID")),
		WeComSecretSet:         strings.TrimSpace(os.Getenv("WECOM_SECRET")) != "",
		WeComTokenSet:          strings.TrimSpace(os.Getenv("WECOM_TOKEN")) != "",
		WeComEncodingAESKeySet: strings.TrimSpace(os.Getenv("WECOM_ENCODING_AES_KEY")) != "",
		WeComAllowedUsers:      strings.TrimSpace(os.Getenv("WECOM_ALLOWED_USERS")),
	}
}

// ApplyGatewaySettings 写入 .env 并更新进程环境变量
func ApplyGatewaySettings(p GatewaySettingsPayload) error {
	current := GatewayFromEnv()

	updates := map[string]string{
		"GATEWAY_ENABLED":    boolStr(p.Enabled || anyPlatformEnabled(p)),
		"GATEWAY_PUBLIC_URL": strings.TrimSpace(p.PublicURL),

		"TELEGRAM_ENABLED": boolStr(p.TelegramEnabled),
		"DINGTALK_ENABLED": boolStr(p.DingTalkEnabled),
		"FEISHU_ENABLED":   boolStr(p.FeishuEnabled),
		"WECOM_ENABLED":    boolStr(p.WeComEnabled),

		"TELEGRAM_ALLOWED_USERS": strings.TrimSpace(p.TelegramAllowedUsers),
		"DINGTALK_CLIENT_ID":     strings.TrimSpace(p.DingTalkClientID),
		"FEISHU_APP_ID":          strings.TrimSpace(p.FeishuAppID),
		"FEISHU_MODE":            normalizeFeishuMode(p.FeishuMode),
		"WECOM_CORP_ID":          strings.TrimSpace(p.WeComCorpID),
		"WECOM_AGENT_ID":         strings.TrimSpace(p.WeComAgentID),
		"WECOM_ALLOWED_USERS":    strings.TrimSpace(p.WeComAllowedUsers),
	}

	if err := mergeSecret(updates, "TELEGRAM_BOT_TOKEN", p.TelegramBotToken, current.Telegram.BotToken); err != nil {
		return err
	}
	if err := mergeSecret(updates, "DINGTALK_CLIENT_SECRET", p.DingTalkClientSecret, current.DingTalk.ClientSecret); err != nil {
		return err
	}
	if err := mergeSecret(updates, "FEISHU_APP_SECRET", p.FeishuAppSecret, current.Feishu.AppSecret); err != nil {
		return err
	}
	if err := mergeSecret(updates, "WECOM_SECRET", p.WeComSecret, current.WeCom.Secret); err != nil {
		return err
	}
	if err := mergeSecret(updates, "WECOM_TOKEN", p.WeComToken, current.WeCom.Token); err != nil {
		return err
	}
	if err := mergeSecret(updates, "WECOM_ENCODING_AES_KEY", p.WeComEncodingAESKey, current.WeCom.EncodingAESKey); err != nil {
		return err
	}

	return UpdateEnvKeys(gatewayEnvPath(), updates)
}

var gatewayEnvFile = DefaultEnvPath

// SetGatewayEnvFile 测试或自定义 .env 路径
func SetGatewayEnvFile(path string) {
	gatewayEnvFile = path
}

func gatewayEnvPath() string {
	if gatewayEnvFile == "" {
		return DefaultEnvPath
	}
	return gatewayEnvFile
}

func mergeSecret(updates map[string]string, key, incoming, current string) error {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		updates[key] = current
		return nil
	}
	if strings.HasPrefix(incoming, "••••") {
		return fmt.Errorf("%s 格式无效", key)
	}
	updates[key] = incoming
	return nil
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func normalizeFeishuMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "webhook") {
		return "webhook"
	}
	return "websocket"
}

func anyPlatformEnabled(p GatewaySettingsPayload) bool {
	return p.TelegramEnabled || p.DingTalkEnabled || p.FeishuEnabled || p.WeComEnabled
}
