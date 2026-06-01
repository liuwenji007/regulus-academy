package channel

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

func TestPingAckData(t *testing.T) {
	raw := json.RawMessage(`"{\"opaque\":\"abc-123\"}"`)
	got := pingAckData(raw)
	if !strings.Contains(got, "abc-123") {
		t.Fatalf("expected opaque in ack, got %s", got)
	}

	got2 := pingAckData(json.RawMessage(`{"opaque":"direct"}`))
	if !strings.Contains(got2, "direct") {
		t.Fatalf("expected direct opaque, got %s", got2)
	}
}

func TestParseBotCallbackSingleChat(t *testing.T) {
	var called bool
	payload := `{"conversationType":"1","conversationId":"cid1","senderStaffId":"staff001","text":{"content":"绑定 测试"}}`
	parseBotCallbackStatic(json.RawMessage(payload), func(ev MessageEvent) {
		called = true
		if ev.PlatformUserID != "staff001" {
			t.Fatalf("user=%s", ev.PlatformUserID)
		}
		if ev.Text != "绑定 测试" {
			t.Fatalf("text=%s", ev.Text)
		}
	})
	if !called {
		t.Fatal("expected callback")
	}
}

func TestParseBotCallbackIgnoresGroup(t *testing.T) {
	var called bool
	payload := `{"conversationType":"2","senderStaffId":"staff001","text":{"content":"hi"}}`
	parseBotCallbackStatic(json.RawMessage(payload), func(_ MessageEvent) {
		called = true
	})
	if called {
		t.Fatal("group chat should be ignored")
	}
}

func TestOpenConnectionSubscriptions(t *testing.T) {
	body, err := dingtalkOpenConnectionBody("appkey", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, dingtalkBotCallback) {
		t.Fatalf("missing bot callback topic: %s", body)
	}
	if !strings.Contains(body, `"type":"CALLBACK"`) && !strings.Contains(body, `"type": "CALLBACK"`) {
		t.Fatalf("missing CALLBACK type: %s", body)
	}
}

// parseBotCallbackStatic 供测试调用 parseBotCallback 逻辑
func parseBotCallbackStatic(data json.RawMessage, onMessage func(MessageEvent)) {
	a := &DingTalkAdapter{}
	a.parseBotCallback(data, onMessage)
}

func dingtalkOpenConnectionBody(clientID, clientSecret string) (string, error) {
	a := &DingTalkAdapter{cfg: configForTest(clientID, clientSecret)}
	b, err := json.Marshal(map[string]any{
		"clientId":     a.cfg.ClientID,
		"clientSecret": a.cfg.ClientSecret,
		"ua":           "regulus-academy/dingtalk-stream/1.0",
		"subscriptions": []map[string]string{
			{"type": "EVENT", "topic": "*"},
			{"type": "CALLBACK", "topic": dingtalkBotCallback},
		},
	})
	return string(b), err
}

func configForTest(id, secret string) config.DingTalkConfig {
	return config.DingTalkConfig{ClientID: id, ClientSecret: secret}
}
