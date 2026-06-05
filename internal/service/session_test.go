package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type seqLLM struct {
	replies []string
	n       int
}

func (m *seqLLM) Configured() bool { return true }
func (m *seqLLM) Name() string     { return "mock" }
func (m *seqLLM) Model() string    { return "mock" }
func (m *seqLLM) Ping(context.Context) error { return nil }

func (m *seqLLM) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return m.ChatWithTemp(ctx, messages, 0.6)
}

func (m *seqLLM) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	if m.n >= len(m.replies) {
		return "ok", nil
	}
	r := m.replies[m.n]
	m.n++
	return r, nil
}

func (m *seqLLM) ChatJSON(context.Context, []llm.Message, float64, any) error {
	return nil
}

func chdirRepo(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "regulus-coach")); err == nil {
			if err := os.Chdir(wd); err != nil {
				t.Fatal(err)
			}
			return
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("找不到 regulus-coach 目录")
		}
		wd = parent
	}
}

func TestStartOrResumeSession_completedNodePreservesProgress(t *testing.T) {
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

	nodeKey := "goroutine_basics"
	_ = store.UpsertProgress(storage.UserProgress{
		UserID: storage.DefaultUserID, DomainID: tree.DomainID,
		NodeKey: nodeKey, Layer: "entry", Status: "completed", Mastery: 0.9,
	})

	llmMock := &seqLLM{replies: []string{"不应调用"}}
	coach, err := agent.NewCoach(store, llmMock)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSessionService(store, coach, llmMock)

	result, err := svc.StartOrResumeSession(context.Background(), storage.DefaultUserID, tree.DomainID, nodeKey, "entry")
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.Phase != "completed" {
		t.Fatalf("phase=%s", result.Session.Phase)
	}
	if result.Content != "" {
		t.Fatalf("review should not begin explain: %q", result.Content)
	}
	if llmMock.n != 0 {
		t.Fatal("completed review should not call LLM")
	}

	progress, err := store.ListProgress(storage.DefaultUserID, tree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range progress {
		if p.NodeKey == nodeKey && p.Status != "completed" {
			t.Fatalf("progress downgraded: %+v", p)
		}
	}
}

func TestSendCoachMessageNextSessionKeepsUserTurn(t *testing.T) {
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

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "completed", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	sess.Status = "completed"
	if err := store.UpdateSession(sess); err != nil {
		t.Fatal(err)
	}

	begin := "下一节开场讲解"
	llmMock := &seqLLM{replies: []string{begin}}
	coach, err := agent.NewCoach(store, llmMock)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSessionService(store, coach, llmMock)

	out, err := svc.SendCoachMessage(context.Background(), storage.DefaultUserID, sess.ID, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if out.Result.NextSessionID == "" || out.Session == nil {
		t.Fatalf("应切换会话 result=%+v session=%v", out.Result, out.Session)
	}
	if out.Session.ID == sess.ID {
		t.Fatal("应返回新会话")
	}

	oldMsgs, err := store.ListMessages(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range oldMsgs {
		if m.Role == "user" && m.Content == "下一节" {
			t.Fatal("触发「下一节」的用户消息不应留在已完成会话")
		}
	}

	newMsgs, err := store.ListMessages(out.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var hasUser, hasAssistant bool
	for _, m := range newMsgs {
		if m.Role == "user" && m.Content == "下一节" {
			hasUser = true
		}
		if m.Role == "assistant" && m.Content == out.Result.Content {
			hasAssistant = true
		}
	}
	if !hasUser {
		t.Fatal("新会话应包含用户触发消息")
	}
	if !hasAssistant {
		t.Fatal("新会话应包含助手回复")
	}
}
