package channel

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

// TelegramAdapter Telegram long polling
type TelegramAdapter struct {
	cfg   config.TelegramConfig
	bot   *tgbotapi.BotAPI
	allow map[string]struct{}
}

// NewTelegramAdapter 创建 Telegram 适配器
func NewTelegramAdapter(cfg config.TelegramConfig) *TelegramAdapter {
	allow := make(map[string]struct{})
	for _, id := range cfg.AllowedUsers {
		allow[id] = struct{}{}
	}
	return &TelegramAdapter{cfg: cfg, allow: allow}
}

func (b *TelegramAdapter) Name() string { return PlatformTelegram }

func (b *TelegramAdapter) Start(ctx context.Context, onMessage func(MessageEvent)) error {
	bot, err := tgbotapi.NewBotAPI(b.cfg.BotToken)
	if err != nil {
		return fmt.Errorf("telegram bot: %w", err)
	}
	b.bot = bot
	log.Printf("[telegram] bot @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			bot.StopReceivingUpdates()
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil || update.Message.Chat == nil {
				continue
			}
			if !update.Message.Chat.IsPrivate() {
				continue
			}
			userID := strconv.FormatInt(update.Message.From.ID, 10)
			if len(b.allow) > 0 {
				if _, ok := b.allow[userID]; !ok {
					continue
				}
			}
			text := strings.TrimSpace(update.Message.Text)
			if text == "" {
				continue
			}
			onMessage(MessageEvent{
				Platform:       PlatformTelegram,
				ChatID:         strconv.FormatInt(update.Message.Chat.ID, 10),
				PlatformUserID: userID,
				Text:           text,
			})
		}
	}
}

func (b *TelegramAdapter) SendText(_ context.Context, target ReplyTarget, text string) error {
	if b.bot == nil {
		return fmt.Errorf("telegram bot not started")
	}
	chatID, err := strconv.ParseInt(target.ChatID, 10, 64)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(chatID, text)
	_, err = b.bot.Send(msg)
	return err
}
