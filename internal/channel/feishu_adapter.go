package channel

import (
	"context"
	"fmt"
	"log"

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
			msgType := ""
			if msg.MessageType != nil {
				msgType = *msg.MessageType
			}
			chatType := "unknown"
			if msg.ChatType != nil {
				chatType = *msg.ChatType
			}
			msgID := ""
			if msg.MessageId != nil {
				msgID = *msg.MessageId
			}
			log.Printf("[feishu] 收到消息事件 id=%s type=%s chat_type=%s", msgID, msgType, chatType)

			if msgType != "text" && msgType != "post" {
				log.Printf("[feishu] 忽略非文本消息 type=%s", msgType)
				return nil
			}
			if msg.ChatType == nil || *msg.ChatType != "p2p" {
				log.Printf("[feishu] 忽略非私聊消息 chat_type=%s（请搜索机器人名称，进入与机器人的单聊窗口发消息）", chatType)
				return nil
			}
			if msg.Content == nil {
				return nil
			}
			text := parseFeishuText(msgType, *msg.Content)
			if text == "" {
				log.Printf("[feishu] 无法解析消息内容 type=%s raw=%s", msgType, truncate(*msg.Content, 120))
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
				log.Println("[feishu] 忽略消息: 缺少 open_id 或 chat_id")
				return nil
			}
			log.Printf("[feishu] 处理私聊消息 from=%s text=%q", openID, truncate(text, 80))
			RecordPlatformEvent(PlatformFeishu)
			ev := MessageEvent{
				Platform:       PlatformFeishu,
				ChatID:         chatID,
				PlatformUserID: openID,
				Text:           text,
			}
			go onMessage(ev)
			return nil
		})

	log.Println("[feishu] WebSocket 长连接启动中…")
	log.Println("[feishu] 提示: 同一 App 同时只应有一个长连接；若重复运行 go run 或 Docker+本地同时启动，消息会随机丢失")
	log.Println("[feishu] 提示: 请保持本服务运行，再到飞书开放平台 → 事件订阅 → 选「使用长连接」→ 添加 im.message.receive_v1 → 保存并发布版本")

	cli := larkws.NewClient(w.cfg.AppID, w.cfg.AppSecret,
		larkws.WithEventHandler(handler),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
		larkws.WithOnReady(func() {
			SetPlatformConnected(PlatformFeishu, true)
			log.Println("[feishu] WebSocket 长连接已就绪（开发者后台「事件订阅」应显示已连接）")
		}),
		larkws.WithOnError(func(err error) {
			RecordPlatformError(PlatformFeishu, err.Error())
			log.Printf("[feishu] WebSocket 错误: %v", err)
		}),
		larkws.WithOnDisconnected(func() {
			SetPlatformConnected(PlatformFeishu, false)
			log.Println("[feishu] WebSocket 已断开，SDK 将自动重连…")
		}),
		larkws.WithOnReconnecting(func() {
			log.Println("[feishu] WebSocket 正在重连…")
		}),
		larkws.WithOnReconnected(func() {
			SetPlatformConnected(PlatformFeishu, true)
			log.Println("[feishu] WebSocket 重连成功")
		}),
	)

	return cli.Start(ctx)
}

func (w *FeishuAdapter) SendText(ctx context.Context, target ReplyTarget, text string) error {
	if err := w.client.sendText(ctx, target.ChatID, text); err != nil {
		return fmt.Errorf("feishu send: %w", err)
	}
	return nil
}
