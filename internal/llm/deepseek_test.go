package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"pong"}}]}`))
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL)
	reply, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "ping"}})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "pong" {
		t.Errorf("期望 pong，得到 %s", reply)
	}
}

func TestChatNoKey(t *testing.T) {
	client := NewClient("", "http://localhost")
	_, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("期望未配置 Key 时返回错误")
	}
}
