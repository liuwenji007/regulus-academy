package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (h *Handler) gatewayInfo(w http.ResponseWriter, r *http.Request) {
	cfg := config.GatewayFromEnv()
	base := publicBaseURL(r)

	bindings, err := h.store.ListChannelBindingsForUser(userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bindings == nil {
		bindings = []storage.ChannelBinding{}
	}

	platforms := buildGatewayPlatforms(cfg, base)
	active := 0
	for _, p := range platforms {
		if p["status"] == "ready" {
			active++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":         cfg.Enabled,
		"activePlatforms": active,
		"publicBaseUrl":   base,
		"platforms":       platforms,
		"bindings":        bindings,
		"commands":        gatewayCommands(),
		"settings":        config.GatewaySettingsViewFromEnv(),
		"needsRestart":    true,
	})
}

func (h *Handler) updateGatewayConfig(w http.ResponseWriter, r *http.Request) {
	var payload config.GatewaySettingsPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if payload.FeishuMode != "" && payload.FeishuMode != "websocket" && payload.FeishuMode != "webhook" {
		writeError(w, http.StatusBadRequest, "FEISHU_MODE 仅支持 websocket 或 webhook")
		return
	}
	if err := config.ApplyGatewaySettings(payload); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.gatewayInfo(w, r)
}

func buildGatewayPlatforms(cfg config.GatewayConfig, base string) []map[string]any {
	out := []map[string]any{
		telegramPlatform(cfg, base),
		dingtalkPlatform(cfg, base),
		feishuPlatform(cfg, base),
		wecomPlatform(cfg, base),
	}
	return out
}

func telegramPlatform(cfg config.GatewayConfig, _ string) map[string]any {
	t := cfg.Telegram
	configured := strings.TrimSpace(t.BotToken) != ""
	return map[string]any{
		"id":          "telegram",
		"name":        "Telegram",
		"enabled":     cfg.Enabled && t.Enabled,
		"configured":  configured,
		"status":      platformStatus(cfg.Enabled && t.Enabled, configured),
		"connection":  "Long Polling（内网可用）",
		"envVars":     []string{"TELEGRAM_BOT_TOKEN", "TELEGRAM_ALLOWED_USERS（可选）"},
		"setupHint":   "通过 @BotFather 创建 Bot 并填入 Token",
	}
}

func dingtalkPlatform(cfg config.GatewayConfig, _ string) map[string]any {
	d := cfg.DingTalk
	configured := strings.TrimSpace(d.ClientID) != "" && strings.TrimSpace(d.ClientSecret) != ""
	return map[string]any{
		"id":         "dingtalk",
		"name":       "钉钉",
		"enabled":    cfg.Enabled && d.Enabled,
		"configured": configured,
		"status":     platformStatus(cfg.Enabled && d.Enabled, configured),
		"connection": "Stream WebSocket（内网可用）",
		"envVars":    []string{"DINGTALK_CLIENT_ID", "DINGTALK_CLIENT_SECRET"},
		"setupHint":  "钉钉开放平台 → 企业内部应用 → Stream 机器人",
	}
}

func feishuPlatform(cfg config.GatewayConfig, base string) map[string]any {
	f := cfg.Feishu
	configured := strings.TrimSpace(f.AppID) != "" && strings.TrimSpace(f.AppSecret) != ""
	mode := f.Mode
	if mode == "" {
		mode = "websocket"
	}
	connection := "WebSocket 长连接（内网可用）"
	webhookURL := ""
	needsHTTPS := false
	if mode == "webhook" {
		connection = "HTTP Webhook（需公网 HTTPS）"
		webhookURL = base + "/webhook/feishu"
		needsHTTPS = true
	}
	p := map[string]any{
		"id":               "feishu",
		"name":             "飞书",
		"enabled":          cfg.Enabled && f.Enabled,
		"configured":       configured,
		"status":           platformStatus(cfg.Enabled && f.Enabled, configured),
		"connection":       connection,
		"mode":             mode,
		"envVars":          []string{"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_MODE=websocket|webhook"},
		"setupHint":        "飞书开放平台 → 企业自建应用 → 事件订阅",
		"needsPublicHttps": needsHTTPS,
	}
	if webhookURL != "" {
		p["webhookUrl"] = webhookURL
	}
	return p
}

func wecomPlatform(cfg config.GatewayConfig, base string) map[string]any {
	w := cfg.WeCom
	configured := strings.TrimSpace(w.Token) != "" &&
		strings.TrimSpace(w.EncodingAESKey) != "" &&
		strings.TrimSpace(w.CorpID) != "" &&
		strings.TrimSpace(w.Secret) != ""
	return map[string]any{
		"id":               "wecom",
		"name":             "企业微信",
		"enabled":          cfg.Enabled && w.Enabled,
		"configured":       configured,
		"status":           platformStatus(cfg.Enabled && w.Enabled, configured),
		"connection":       "HTTP 回调（需公网 HTTPS）",
		"webhookUrl":       base + "/webhook/wecom",
		"needsPublicHttps": true,
		"envVars": []string{
			"WECOM_CORP_ID", "WECOM_AGENT_ID", "WECOM_SECRET",
			"WECOM_TOKEN", "WECOM_ENCODING_AES_KEY", "WECOM_ALLOWED_USERS（可选）",
		},
		"setupHint": "企业微信管理后台 → 应用 → 接收消息 → 填入回调 URL",
	}
}

func platformStatus(enabled, configured bool) string {
	if !enabled {
		return "off"
	}
	if !configured {
		return "pending"
	}
	return "ready"
}

func gatewayCommands() []map[string]string {
	return []map[string]string{
		{"command": "绑定 角色名", "description": "绑定 Web 端学习角色（需先在 Web 创建）"},
		{"command": "课程", "description": "查看知识库列表"},
		{"command": "学习 1", "description": "查看某门课程的节点"},
		{"command": "节点 1", "description": "开始或继续某节点学习"},
		{"command": "继续", "description": "查看当前学习状态"},
		{"command": "进度", "description": "查看学习进度"},
		{"command": "帮助", "description": "显示命令列表"},
	}
}

func publicBaseURL(r *http.Request) string {
	if v := strings.TrimSpace(os.Getenv("GATEWAY_PUBLIC_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	return scheme + "://" + host
}
