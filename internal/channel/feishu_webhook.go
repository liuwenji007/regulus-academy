package channel

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

// FeishuWebhook 飞书 HTTP 事件回调（无需官方 SDK）
// 在飞书开发者后台选择「将事件发送至开发者服务器」，请求 URL 填 https://你的域名/webhook/feishu
type FeishuWebhook struct {
	cfg    config.FeishuConfig
	client *feishuClient
	router *Router
}

// NewFeishuWebhook 创建飞书 webhook
func NewFeishuWebhook(cfg config.FeishuConfig, router *Router) *FeishuWebhook {
	return &FeishuWebhook{
		cfg:    cfg,
		client: newFeishuClient(cfg),
		router: router,
	}
}

func (w *FeishuWebhook) Name() string { return PlatformFeishu }

func (w *FeishuWebhook) SendText(ctx context.Context, target ReplyTarget, text string) error {
	return w.client.sendText(ctx, target.ChatID, text)
}

// Start webhook 模式由 HTTP 驱动，不启动长连接
func (w *FeishuWebhook) Start(ctx context.Context, _ func(MessageEvent)) error {
	<-ctx.Done()
	return ctx.Err()
}

// Handle 处理飞书回调（URL 验证 + 事件）
func (w *FeishuWebhook) Handle(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(rw, "read body", http.StatusBadRequest)
		return
	}

	// URL 验证
	var challenge struct {
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(body, &challenge); err == nil && challenge.Type == "url_verification" {
		if !w.verifyToken(challenge.Token) {
			http.Error(rw, "invalid token", http.StatusForbidden)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{"challenge": challenge.Challenge})
		return
	}

	// 事件 2.0
	var envelope struct {
		Schema string `json:"schema"`
		Header struct {
			EventType string `json:"event_type"`
			Token     string `json:"token"`
		} `json:"header"`
		Event struct {
			Message struct {
				ChatID      string `json:"chat_id"`
				MessageType string `json:"message_type"`
				Content     string `json:"content"`
				ChatType    string `json:"chat_type"`
			} `json:"message"`
			Sender struct {
				SenderID struct {
					OpenID string `json:"open_id"`
				} `json:"sender_id"`
			} `json:"sender"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		log.Printf("[feishu] 无法解析事件: %v", err)
		rw.WriteHeader(http.StatusOK)
		return
	}

	if !w.verifyToken(envelope.Header.Token) {
		http.Error(rw, "invalid token", http.StatusForbidden)
		return
	}

	if envelope.Header.EventType != "im.message.receive_v1" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	msg := envelope.Event.Message
	if msg.MessageType != "text" || msg.ChatType != "p2p" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(msg.Content), &content); err != nil {
		rw.WriteHeader(http.StatusOK)
		return
	}
	text := strings.TrimSpace(content.Text)
	openID := envelope.Event.Sender.SenderID.OpenID
	if text == "" || openID == "" || msg.ChatID == "" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	ev := MessageEvent{
		Platform:       PlatformFeishu,
		ChatID:         msg.ChatID,
		PlatformUserID: openID,
		Text:           text,
	}
	RecordPlatformEvent(PlatformFeishu)

	target := ReplyFromEvent(ev)
	replies := w.router.Handle(r.Context(), ev)
	all := append(replies.InstantReplies, replies.Replies...)
	if len(all) > 0 {
		Deliver(r.Context(), w, target, all)
	}
	rw.WriteHeader(http.StatusOK)
}

// verifyToken 校验飞书 Verification Token；未配置 FEISHU_VERIFY_TOKEN 时跳过（向后兼容）
func (w *FeishuWebhook) verifyToken(token string) bool {
	expected := strings.TrimSpace(w.cfg.VerifyToken)
	if expected == "" {
		return true
	}
	return strings.TrimSpace(token) == expected
}
