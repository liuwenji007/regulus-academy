package channel

import (
	"encoding/json"
	"regexp"
	"strings"
)

var feishuMentionRe = regexp.MustCompile(`@_user_\d+\s*`)

// parseFeishuText 从飞书消息 content 提取纯文本（支持 text / post）
func parseFeishuText(messageType, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch messageType {
	case "text":
		var content struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(raw), &content); err != nil {
			return ""
		}
		return normalizeFeishuText(content.Text)
	case "post":
		var content struct {
			Content [][]struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal([]byte(raw), &content); err != nil {
			return ""
		}
		var b strings.Builder
		for _, row := range content.Content {
			for _, seg := range row {
				if seg.Tag == "text" && seg.Text != "" {
					b.WriteString(seg.Text)
				}
			}
		}
		return normalizeFeishuText(b.String())
	default:
		return ""
	}
}

func normalizeFeishuText(s string) string {
	s = feishuMentionRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
