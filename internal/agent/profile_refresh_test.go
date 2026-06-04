package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestFormatTranscriptForProfile(t *testing.T) {
	msgs := []storage.SessionMessage{
		{Role: "assistant", Content: "开场"},
		{Role: "user", Content: "我是前端出身"},
		{Role: "assistant", Content: "补讲架构"},
	}
	got := formatTranscriptForProfile(msgs)
	if !containsAll(got, "用户", "前端出身", "教练") {
		t.Fatalf("transcript=%q", got)
	}
}

func TestRefreshUserProfileSkipsWithoutUserMessages(t *testing.T) {
	chdirToRepo(t)
	store, err := storage.Open(t.TempDir() + "/prof.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	mp := &mockProvider{replies: []string{mustJSON(ProfileRefreshOutput{Summary: "不应写入"})}}
	coach, err := NewCoach(store, mp)
	if err != nil {
		t.Fatal(err)
	}

	user, err := store.CreateUser("画像测试")
	if err != nil {
		t.Fatal(err)
	}
	reg := domain.NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := store.CreateSession(user.ID, tree.DomainID, "go-concurrency", "goroutine_basics", "completed", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(sess.ID, "assistant", "仅教练开场")

	if err := coach.RefreshUserProfileAfterNode(context.Background(), sess, nil); err != nil {
		t.Fatal(err)
	}
	if mp.calls != 0 {
		t.Fatalf("无用户发言时不应调用 LLM: calls=%d", mp.calls)
	}
}

func TestRefreshUserProfileUpdatesSummary(t *testing.T) {
	chdirToRepo(t)
	store, err := storage.Open(t.TempDir() + "/prof2.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	newSummary := "前端开发背景，偏好先建立整体结构再练题"
	cm := &mockProvider{replies: []string{
		mustJSON(ProfileRefreshOutput{Summary: newSummary}),
	}}
	coach, err := NewCoach(store, cm)
	if err != nil {
		t.Fatal(err)
	}

	user, err := store.CreateUser("画像测试")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpdateUserProfileSummary(user.ID, "旧画像")

	reg := coach.registry
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", tree, string(nodesJSON), storage.DomainSourceSkillPack)
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(user.ID, tree.DomainID, "go-concurrency", "goroutine_basics", "completed", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(sess.ID, "assistant", "讲解")
	_, _ = store.AddMessage(sess.ID, "user", "我是前端开发")

	err = coach.RefreshUserProfileAfterNode(context.Background(), sess, &storage.SessionContext{RecentMistakes: []string{"channel"}})
	if err != nil {
		t.Fatal(err)
	}
	u, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.ProfileSummary != newSummary {
		t.Fatalf("profile=%q", u.ProfileSummary)
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
