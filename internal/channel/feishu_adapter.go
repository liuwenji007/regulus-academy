package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

// FeishuAdapter 飞书官方 SDK WebSocket 长连接（无需公网回调）
type FeishuAdapter struct {
	cfg    config.FeishuConfig
	client *feishuClient
}

// NewFeishuAdapter 创建飞书长连接适配器
func NewFeishuAdapter(cfg config.FeishuConfig) *FeishuAdapter {
	return &FeishuAdapter{
		cfg:    cfg,
		client: newFeishuClient(cfg),
	}
}

func (w *FeishuAdapter) Name() string { return PlatformFeishu }

func (w *FeishuAdapter) Start(ctx context.Context, onMessage func(MessageEvent)) error {
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			if event == nil || event.Event == nil || event.Event.Message == nil {
				return nil
			}
			msg := event.Event.Message
			if msg.MessageType == nil || *msg.MessageType != "text" {
				return nil
			}
			if msg.ChatType == nil || *msg.ChatType != "p2p" {
				return nil
			}
			if msg.Content == nil {
				return nil
			}
			var content struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(*msg.Content), &content); err != nil {
				return nil
			}
			text := strings.TrimSpace(content.Text)
			if text == "" {
				return nil
			}
			openID := ""
			if event.Event.Sender != nil && event.Event.Sender.SenderId != nil && event.Event.Sender.SenderId.OpenId != nil {
				openID = *event.Event.Sender.SenderId.OpenId
			}
			chatID := ""
			if msg.ChatId != nil {
				chatID = *msg.ChatId
			}
			if openID == "" || chatID == "" {
				return nil
			}
			onMessage(MessageEvent{
				Platform:       PlatformFeishu,
				ChatID:         chatID,
				PlatformUserID: openID,
				Text:           text,
			})
			return nil
		})

	cli := larkws.NewClient(w.cfg.AppID, w.cfg.AppSecret,
		larkws.WithEventHandler(handler),
		larkws.WithLogLevel(larkcore.LogLevelError),
	)

	log.Println("[feishu] WebSocket 长连接启动中…（开发者后台需选「使用长连接接收事件」）")
	errCh := make(chan error, 1)
	go func() {
		errCh <- cli.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (w *FeishuAdapter) SendText(ctx context.Context, target ReplyTarget, text string) error {
	if err := w.client.sendText(ctx, target.ChatID, text); err != nil {
		return fmt.Errorf("feishu send: %w", err)
	}
	return nil
}
