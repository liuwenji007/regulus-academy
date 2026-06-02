package channel

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type navLLMMock struct {
	response string
}

func (m *navLLMMock) Name() string  { return "mock" }
func (m *navLLMMock) Model() string { return "mock" }
func (m *navLLMMock) Configured() bool {
	return m.response != ""
}
func (m *navLLMMock) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return m.response, nil
}
func (m *navLLMMock) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	return m.response, nil
}
func (m *navLLMMock) ChatJSON(ctx context.Context, messages []llm.Message, temp float64, dest any) error {
	return json.Unmarshal([]byte(m.response), dest)
}
func (m *navLLMMock) Ping(ctx context.Context) error { return nil }

func setupNavRouter(t *testing.T, llmClient llm.Provider) (*Router, *storage.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	if _, _, err := store.CreateDomain("Go 并发"); err != nil {
		t.Fatal(err)
	}
	userID := storage.DefaultUserID

	coach, err := agent.NewCoach(store, llmClient)
	if err != nil {
		t.Fatal(err)
	}
	sessions := service.NewSessionService(store, coach, llmClient)
	router := NewRouter(store, sessions, llmClient)

	_ = store.UpsertChannelBinding(PlatformTelegram, "u-nav", userID, "测试")
	return router, store, userID
}

func TestRouterNaturalLanguageCourses(t *testing.T) {
	llmClient := llm.NewClient("test", "http://localhost")
	router, _, _ := setupNavRouter(t, llmClient)

	result := router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "我的课程",
	})
	if len(result.Replies) == 0 || !strings.Contains(result.Replies[0], "Go 并发") {
		t.Fatalf("courses: %v", result.Replies)
	}
}

func TestRouterNaturalLanguageLearn(t *testing.T) {
	llmClient := llm.NewClient("test", "http://localhost")
	router, _, _ := setupNavRouter(t, llmClient)

	result := router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "学 Go 并发",
	})
	if len(result.Replies) == 0 || !strings.Contains(result.Replies[0], "节点列表") {
		t.Fatalf("learn: %v", result.Replies)
	}
}

func TestRouterLLMNavIntent(t *testing.T) {
	mock := &navLLMMock{response: `{"action":"list_courses","course_ref":"","node_ref":"","reply_hint":""}`}
	router, _, _ := setupNavRouter(t, mock)

	result := router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "帮我看看课",
	})
	if len(result.Replies) == 0 || !strings.Contains(result.Replies[0], "Go 并发") {
		t.Fatalf("llm nav: %v", result.Replies)
	}
}

func TestRouterLearningQuestionGoesCoach(t *testing.T) {
	llmClient := &navLLMMock{response: `{"action":"list_courses","course_ref":"","node_ref":"","reply_hint":""}`}
	router, store, userID := setupNavRouter(t, llmClient)

	domains, _ := store.ListDomainSummaries(userID)
	domainID := domains[0].ID
	tree, _ := store.GetDomainTree(userID, domainID)
	nodes := flattenNodes(tree)

	_, err := store.CreateSession(userID, domainID, "", nodes[0].Key, "explain", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.SetChannelActiveNode(userID, domainID, nodes[0].Key)

	result := router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "一般go标准项目里需要哪些",
	})
	if len(result.Replies) == 0 {
		t.Fatal("expected reply")
	}
	if strings.Contains(result.Replies[0], "你的课程：") {
		t.Fatalf("不应返回课表，应走 Coach: %q", result.Replies[0])
	}
}

func TestRouterContinueWithSessionGoesCoach(t *testing.T) {
	llmClient := llm.NewClient("test", "http://localhost")
	router, store, userID := setupNavRouter(t, llmClient)

	domains, _ := store.ListDomainSummaries(userID)
	if len(domains) == 0 {
		t.Fatal("no domain")
	}
	domainID := domains[0].ID
	tree, _ := store.GetDomainTree(userID, domainID)
	nodes := flattenNodes(tree)
	if len(nodes) == 0 {
		t.Fatal("no nodes")
	}

	_, err := store.CreateSession(userID, domainID, "", nodes[0].Key, "explain", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.SetChannelActiveNode(userID, domainID, nodes[0].Key)

	isCoach := router.IsCoachMessage(MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "接着学",
	})
	if !isCoach {
		t.Fatal("接着学 + active session 应走 Coach")
	}
}
