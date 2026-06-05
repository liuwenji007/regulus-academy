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

func TestStartNextNodeCreatesFreshSessionDespiteOlderIncomplete(t *testing.T) {
	chdirRepo(t)

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
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack, false)
	if err != nil {
		t.Fatal(err)
	}

	nextKey, _, _, ok := domain.NextNodeAfter(tree, "goroutine_basics")
	if !ok {
		t.Fatal("应有下一节点")
	}

	stale, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", nextKey, "exercise", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(stale.ID, "assistant", "旧会话里的题目")

	done, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "completed", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	done.Status = "completed"
	if err := store.UpdateSession(done); err != nil {
		t.Fatal(err)
	}

	llmMock := &seqLLM{replies: []string{"新一节开场"}}
	coach, err := agent.NewCoach(store, llmMock)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSessionService(store, coach, llmMock)

	result, err := svc.StartNextNode(context.Background(), storage.DefaultUserID, done.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.ID == stale.ID || result.Session.ID == done.ID {
		t.Fatalf("应创建新会话 got=%s stale=%s done=%s", result.Session.ID, stale.ID, done.ID)
	}
	if result.Session.NodeKey != nextKey {
		t.Fatalf("nodeKey=%s want %s", result.Session.NodeKey, nextKey)
	}
	if result.Resumed {
		t.Fatal("不应标记为 resumed")
	}
	msgs, err := store.ListMessages(result.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Role != "assistant" {
		t.Fatalf("新会话应只有开场助手消息 msgs=%v", msgs)
	}
}

func TestStartNextNodeWhenProgressCompletedButPhaseStale(t *testing.T) {
	chdirRepo(t)

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
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack, false)
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "review", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpsertProgress(storage.UserProgress{
		UserID:   storage.DefaultUserID,
		DomainID: tree.DomainID,
		NodeKey:  "goroutine_basics",
		Layer:    "entry",
		Status:   "completed",
		Mastery:  0.8,
	})

	llmMock := &seqLLM{replies: []string{"下一节开场"}}
	coach, err := agent.NewCoach(store, llmMock)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSessionService(store, coach, llmMock)

	result, err := svc.StartNextNode(context.Background(), storage.DefaultUserID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.NodeKey != "first_goroutine" {
		t.Fatalf("nodeKey=%s", result.Session.NodeKey)
	}
	updated, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Phase != "completed" {
		t.Fatalf("应回填 completed phase=%s", updated.Phase)
	}
}
