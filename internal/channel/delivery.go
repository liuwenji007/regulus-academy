package channel

import (
	"context"
	"log"
	"time"
)

const deliveryRetryDelay = 400 * time.Millisecond

// Deliver 统一出站：分片发送，失败重试一次
func Deliver(ctx context.Context, adapter Adapter, target ReplyTarget, replies []string) {
	if len(replies) == 0 {
		return
	}
	name := adapter.Name()
	for i, reply := range replies {
		chunks := SplitMessage(reply, defaultChunkRunes)
		for j, chunk := range chunks {
			if err := sendWithRetry(ctx, adapter, target, chunk); err != nil {
				log.Printf("[delivery/%s] 发送失败 reply=%d chunk=%d err=%v", name, i+1, j+1, err)
				RecordPlatformError(name, err.Error())
			} else {
				log.Printf("[delivery/%s] 已发送 reply=%d chunk=%d", name, i+1, j+1)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

func sendWithRetry(ctx context.Context, adapter Adapter, target ReplyTarget, text string) error {
	err := adapter.SendText(ctx, target, text)
	if err == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(deliveryRetryDelay):
	}
	return adapter.SendText(ctx, target, text)
}
