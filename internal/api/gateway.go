package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/channel"
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
		"runtime": map[string]any{
			"platformHealth": channel.AllPlatformHealth(),
		},
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
	platformOn := config.EnvBool("TELEGRAM_ENABLED", true)
	configured := strings.TrimSpace(t.BotToken) != ""
	return map[string]any{
		"id":              "telegram",
		"name":            "Telegram",
		"platformEnabled": platformOn,
		"enabled":         cfg.Enabled && platformOn,
		"configured":      configured,
		"status":          platformStatus(cfg.Enabled, platformOn, configured),
		"connection":      "Long Polling（内网可用）",
		"envVars":         []string{"TELEGRAM_BOT_TOKEN", "TELEGRAM_ALLOWED_USERS（可选）"},
		"setupHint":       "通过 @BotFather 创建 Bot 并填入 Token",
	}
}

func dingtalkPlatform(cfg config.GatewayConfig, _ string) map[string]any {
	d := cfg.DingTalk
	platformOn := config.EnvBool("DINGTALK_ENABLED", true)
	configured := strings.TrimSpace(d.ClientID) != "" && strings.TrimSpace(d.ClientSecret) != ""
	return map[string]any{
		"id":              "dingtalk",
		"name":            "钉钉",
		"platformEnabled": platformOn,
		"enabled":         cfg.Enabled && platformOn,
		"configured":      configured,
		"status":          platformStatus(cfg.Enabled, platformOn, configured),
		"connection":      "Stream WebSocket（内网可用）",
		"envVars":         []string{"DINGTALK_CLIENT_ID", "DINGTALK_CLIENT_SECRET"},
		"setupHint":       "钉钉开放平台 → 企业内部应用 → Stream 机器人",
	}
}

func feishuPlatform(cfg config.GatewayConfig, base string) map[string]any {
	f := cfg.Feishu
	platformOn := config.EnvBool("FEISHU_ENABLED", true)
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
		"platformEnabled":  platformOn,
		"enabled":          cfg.Enabled && platformOn,
		"configured":       configured,
		"status":           platformStatus(cfg.Enabled, platformOn, configured),
		"connection":       connection,
		"mode":             mode,
		"envVars":          []string{"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_MODE=websocket|webhook", "FEISHU_ALLOWED_USERS（可选）"},
		"setupHint":        "飞书开放平台 → 企业自建应用 → 事件与回调",
		"needsPublicHttps": needsHTTPS,
	}
	if mode == "websocket" {
		p["setupSteps"] = []string{
			"应用能力 → 开启「机器人」",
			"权限管理 → 开通 im:message、im:message.p2p_msg:readonly、im:message:send_as_bot",
			"先启动本服务（保持运行），再到「事件与回调」→ 选「使用长连接接收事件」",
			"添加事件 im.message.receive_v1 并保存（后台应显示已连接）",
			"版本管理与发布 → 创建版本并发布到企业",
			"在飞书中搜索机器人名称，进入单聊（非群聊）发送：绑定 你的Web角色名",
		}
	}
	if webhookURL != "" {
		p["webhookUrl"] = webhookURL
	}
	health := channel.GetPlatformHealth("feishu")
	p["runtime"] = map[string]any{
		"connected":   health.Connected,
		"lastEventAt": formatTimePtr(health.LastEventAt),
		"lastError":   health.LastError,
	}
	return p
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func wecomPlatform(cfg config.GatewayConfig, base string) map[string]any {
	w := cfg.WeCom
	platformOn := config.EnvBool("WECOM_ENABLED", false)
	configured := strings.TrimSpace(w.Token) != "" &&
		strings.TrimSpace(w.EncodingAESKey) != "" &&
		strings.TrimSpace(w.CorpID) != "" &&
		strings.TrimSpace(w.Secret) != ""
	return map[string]any{
		"id":               "wecom",
		"name":             "企业微信",
		"platformEnabled":  platformOn,
		"enabled":          cfg.Enabled && platformOn,
		"configured":       configured,
		"status":           platformStatus(cfg.Enabled, platformOn, configured),
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

func platformStatus(gatewayEnabled, platformEnabled, configured bool) string {
	if !platformEnabled {
		return "disabled"
	}
	if !configured {
		return "pending"
	}
	if !gatewayEnabled {
		return "waiting"
	}
	return "ready"
}

func gatewayCommands() []map[string]string {
	return []map[string]string{
		{"command": "绑定 角色名", "description": "绑定 Web 端学习角色或 6 位绑定码"},
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
