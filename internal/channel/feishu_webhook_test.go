package channel

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

func TestFeishuWebhookVerifyTokenOptional(t *testing.T) {
	w := NewFeishuWebhook(config.FeishuConfig{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhook/feishu", bytes.NewReader([]byte(`{
		"schema":"2.0",
		"header":{"event_type":"other.event","token":"anything"},
		"event":{}
	}`)))
	rec := httptest.NewRecorder()
	w.Handle(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatalf("未配置 token 时不应拒绝请求，status=%d", rec.Code)
	}
}

func TestFeishuWebhookVerifyTokenMatches(t *testing.T) {
	w := NewFeishuWebhook(config.FeishuConfig{VerifyToken: "expected"}, nil)
	if !w.verifyToken("expected") {
		t.Fatal("匹配 token 应通过")
	}
	if w.verifyToken("wrong") {
		t.Fatal("错误 token 应拒绝")
	}
}

func TestFeishuWebhookRejectsInvalidToken(t *testing.T) {
	w := NewFeishuWebhook(config.FeishuConfig{VerifyToken: "expected"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhook/feishu", bytes.NewReader([]byte(`{
		"schema":"2.0",
		"header":{"event_type":"im.message.receive_v1","token":"wrong"},
		"event":{"message":{"chat_id":"c","message_type":"text","content":"{\"text\":\"hi\"}","chat_type":"p2p"},"sender":{"sender_id":{"open_id":"ou_x"}}}
	}`)))
	rec := httptest.NewRecorder()
	w.Handle(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("错误 token 应返回 403，status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFeishuWebhookURLVerificationToken(t *testing.T) {
	w := NewFeishuWebhook(config.FeishuConfig{VerifyToken: "expected"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhook/feishu", bytes.NewReader([]byte(`{
		"type":"url_verification","token":"wrong","challenge":"abc"
	}`)))
	rec := httptest.NewRecorder()
	w.Handle(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("URL 验证 token 错误应返回 403，status=%d", rec.Code)
	}
}
