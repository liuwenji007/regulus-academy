package channel

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRouterBindAndReject(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("小明")
	if err != nil {
		t.Fatal(err)
	}

	llmClient := llm.NewClient("test", "http://localhost")
	coach, err := agent.NewCoach(store, llmClient)
	if err != nil {
		t.Fatal(err)
	}
	sessions := service.NewSessionService(store, coach, llmClient)
	router := NewRouter(store, sessions)

	ev := MessageEvent{Platform: PlatformTelegram, PlatformUserID: "u1", ChatID: "c1", Text: "你好"}
	result := router.Handle(context.Background(), ev)
	if len(result.Replies) == 0 || !strings.Contains(result.Replies[0], "绑定") {
		t.Fatalf("expected bind prompt, got %v", result.Replies)
	}

	result = router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u1", ChatID: "c1", Text: "绑定 小明",
	})
	if len(result.Replies) == 0 || !strings.Contains(result.Replies[0], "已绑定") {
		t.Fatalf("bind reply: %v", result.Replies)
	}

	b, _ := store.GetChannelBinding(PlatformTelegram, "u1")
	if b == nil || b.UserID != user.ID {
		t.Fatalf("binding not saved: %+v", b)
	}
}
