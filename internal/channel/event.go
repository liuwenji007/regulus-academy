package channel

// Platform 标识
const (
	PlatformTelegram = "telegram"
	PlatformDingTalk = "dingtalk"
	PlatformFeishu   = "feishu"
	PlatformWeCom    = "wecom"
)

// MessageEvent 归一化入站消息
type MessageEvent struct {
	Platform       string
	ChatID         string
	PlatformUserID string
	Text           string
	ReplyTo        ReplyTarget
}

// ReplyTarget 出站回复目标
type ReplyTarget struct {
	Platform       string
	ChatID         string
	PlatformUserID string
}

// ReplyFromEvent 从入站事件构造回复目标
func ReplyFromEvent(ev MessageEvent) ReplyTarget {
	return ReplyTarget{
		Platform:       ev.Platform,
		ChatID:         ev.ChatID,
		PlatformUserID: ev.PlatformUserID,
	}
}
