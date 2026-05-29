package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type mockProvider struct {
	replies []string
	calls   int
}

func (m *mockProvider) Configured() bool { return true }
func (m *mockProvider) Name() string     { return "mock" }
func (m *mockProvider) Model() string    { return "mock" }

func (m *mockProvider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return m.ChatWithTemp(ctx, messages, 0.6)
}

func (m *mockProvider) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	if m.calls >= len(m.replies) {
		return "ok", nil
	}
	r := m.replies[m.calls]
	m.calls++
	return r, nil
}

func (m *mockProvider) ChatJSON(ctx context.Context, messages []llm.Message, temp float64, dest any) error {
	raw, err := m.ChatWithTemp(ctx, messages, temp)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (m *mockProvider) Ping(ctx context.Context) error { return nil }

func chdirToRepo(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			if err := os.Chdir(d); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(wd) })
			return
		}
	}
	t.Fatal("找不到 regulus-coach 目录")
}

func setupCoach(t *testing.T, replies ...string) (*Coach, *storage.Store, *storage.Session) {
	t.Helper()
	chdirToRepo(t)
	store, err := storage.Open(filepath.Join(t.TempDir(), "coach_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	coach, err := NewCoach(store, &mockProvider{replies: replies})
	if err != nil {
		t.Fatal(err)
	}

	reg := domain.NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree("Go 并发", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack)
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	return coach, store, sess
}

func TestHandleMessageExerciseBackToExplain(t *testing.T) {
	coach, store, sess := setupCoach(t, "我们重新讲一下")
	sess.Phase = "exercise"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "不懂")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "explain" {
		t.Fatalf("phase=%s", result.Phase)
	}
}

func TestHandleMessageStartExerciseJSON(t *testing.T) {
	exerciseJSON := `{"question":"写一个 goroutine","question_type":"code","reinforced_concepts":[]}`
	coach, _, sess := setupCoach(t, exerciseJSON)

	result, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Content == "" {
		t.Fatal("期望有题目内容")
	}
}
