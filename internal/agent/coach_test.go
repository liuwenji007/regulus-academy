package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack)
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
	exerciseJSON := `{"question":"写一个 goroutine","question_type":"code_fill","answer_format":"json","reinforced_concepts":[]}`
	coach, store, sess := setupCoach(t, exerciseJSON)

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
	if result.Exercise == nil || result.Exercise.AnswerFormat != "json" {
		t.Fatalf("exercise meta=%+v", result.Exercise)
	}
	sctx := storage.ParseSessionContext(sess)
	if sctx.Exercise == nil || sctx.Exercise.AnswerFormat != "json" {
		t.Fatalf("stored exercise=%+v", sctx.Exercise)
	}
	_ = store
}

func TestEvaluateMasterySkipNotReadyThenForce(t *testing.T) {
	notReady := `{"ready":false,"feedback":"依赖顺序还没讲清","gap_concepts":["任务依赖排序","调试设备前置条件"]}`
	coach, store, sess := setupCoach(t, notReady)

	sess.Phase = "review"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "我已经掌握了，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "review" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.NodeCompleted {
		t.Fatal("不应直接完成")
	}
	sctx := storage.ParseSessionContext(sess)
	if !sctx.SkipMasteryWarned {
		t.Fatal("应标记已提醒")
	}

	result, err = coach.HandleMessage(context.Background(), sess, "我已经掌握了，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("应强制完成 result=%+v", result)
	}
	mistakes, err := store.ListMistakesForReinforce(storage.DefaultUserID, sess.DomainID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mistakes) == 0 {
		t.Fatal("应记录易错概念")
	}
}

func TestEvaluateMasterySkipReady(t *testing.T) {
	ready := `{"ready":true,"feedback":"掌握不错，可以进入下一节","gap_concepts":[]}`
	coach, _, sess := setupCoach(t, ready)

	result, err := coach.HandleMessage(context.Background(), sess, "已经掌握，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("应直接完成 result=%+v", result)
	}
}

func TestStartNextNodeFromExplainPhase(t *testing.T) {
	beginReply := "下一节开场"
	coach, store, sess := setupCoach(t, beginReply)
	if sess.Phase != "explain" {
		t.Fatalf("phase=%s want explain", sess.Phase)
	}

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "explain" || result.NextSessionID == "" {
		t.Fatalf("应切到下一节 result=%+v", result)
	}
	if result.NextNodeKey != "first_goroutine" {
		t.Fatalf("nextNodeKey=%q", result.NextNodeKey)
	}
	newSess, err := store.GetSession(result.NextSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if newSess.NodeKey != "first_goroutine" {
		t.Fatalf("new session=%+v", newSess)
	}
}

func TestBlockStartNextBeforeCompleted(t *testing.T) {
	coach, _, sess := setupCoach(t)
	sess.Phase = "review"
	_ = coach.store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.NextSessionID != "" {
		t.Fatalf("未完成节点不应直接切节: %+v", result)
	}
	if result.Phase != "review" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Content == "" || !containsAll(result.Content, "尚未完成", "已经掌握") {
		t.Fatalf("应提示先完成或申请掌握: %q", result.Content)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestStartNextNodeAfterCompleted(t *testing.T) {
	beginReply := "这是下一节开场讲解"
	coach, store, sess := setupCoach(t, beginReply)
	sess.Phase = "completed"
	sess.Status = "completed"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "explain" || result.NextSessionID == "" {
		t.Fatalf("应进入下一节 result=%+v", result)
	}
	if result.NextNodeKey != "first_goroutine" {
		t.Fatalf("nextNodeKey=%q", result.NextNodeKey)
	}
	newSess, err := store.GetSession(result.NextSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if newSess.NodeKey != "first_goroutine" || newSess.Phase != "explain" {
		t.Fatalf("new session=%+v", newSess)
	}
}
