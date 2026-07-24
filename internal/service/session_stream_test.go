package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestSendCoachMessageStream_persistsAfterClientCancel(t *testing.T) {
	chdirRepo(t)
	t.Setenv("LANGFUSE_ENABLED", "false")

	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	reg := domain.NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}

	reply := "这是流式讲解内容"
	llmMock := &seqLLM{replies: []string{reply}}
	coach, err := agent.NewCoach(store, llmMock)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSessionService(store, coach, llmMock)

	parent, cancel := context.WithCancel(context.Background())
	cancel() // 模拟客户端已断开

	var events []CoachStreamEvent
	out, err := svc.SendCoachMessageStream(parent, storage.DefaultUserID, sess.ID, "什么是 goroutine？", func(ev CoachStreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Result == nil || out.Result.Content != reply {
		t.Fatalf("result=%+v", out.Result)
	}

	msgs, err := store.ListMessages(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var hasUser, hasAssistant bool
	for _, m := range msgs {
		if m.Role == "user" && m.Content == "什么是 goroutine？" {
			hasUser = true
		}
		if m.Role == "assistant" && m.Content == reply {
			hasAssistant = true
		}
	}
	if !hasUser || !hasAssistant {
		t.Fatalf("断连后仍应落库 messages=%+v", msgs)
	}

	var sawDelta, sawMessage bool
	for _, ev := range events {
		if ev.Type == "delta" && ev.Text != "" {
			sawDelta = true
		}
		if ev.Type == "message" && ev.Message != nil {
			sawMessage = true
		}
	}
	if !sawDelta || !sawMessage {
		t.Fatalf("应收到 delta 与 message 事件: %+v", events)
	}
}
