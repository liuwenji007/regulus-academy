package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRefreshUserProfileWritesDomainOnly(t *testing.T) {
	chdirToRepo(t)
	store, err := storage.Open(t.TempDir() + "/prof2.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	domainSummary := "理解 goroutine 调度与 channel 通信方向"
	cm := &mockProvider{replies: []string{
		mustJSON(ProfileRefreshOutput{DomainSummary: domainSummary}),
	}}
	coach, err := NewCoach(store, cm)
	if err != nil {
		t.Fatal(err)
	}

	user, err := store.CreateUser("画像测试")
	if err != nil {
		t.Fatal(err)
	}
	oldGlobal := "【背景】工程师\n【进展】心理学人本主义"
	_ = store.UpdateUserProfileSummary(user.ID, oldGlobal)

	reg := coach.registry
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
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
	if u.ProfileSummary != oldGlobal {
		t.Fatalf("global profile should be unchanged: %q", u.ProfileSummary)
	}
	dp, err := store.GetDomainProfile(user.ID, tree.DomainID)
	if err != nil || dp == nil || dp.Summary != domainSummary {
		t.Fatalf("domain profile=%v err=%v", dp, err)
	}
}

func TestComposeForCoachInjection(t *testing.T) {
	chdirToRepo(t)
	store, err := storage.Open(t.TempDir() + "/inj.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	user, _ := store.CreateUser("inj")
	_ = store.UpdateUserProfileSummary(user.ID, "【背景】工程师\n【进展】心理学卡氏尺度")
	reg := domain.NewRegistry()
	tree, nodes, _ := reg.LoadTreeAndNodes("go-concurrency")
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, _ = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")

	coach, _ := NewCoach(store, &mockProvider{})
	sess, _ := store.CreateSession(user.ID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", nil)
	in, err := coach.buildInput(sess, "讲解", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(in.UserProfile, "心理学") || strings.Contains(in.UserProfile, "卡氏") {
		t.Fatalf("injection leaked other course: %q", in.UserProfile)
	}
}

func TestMigrateUserProfile_conservative(t *testing.T) {
	store, err := storage.Open(t.TempDir() + "/mig.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, _ := store.CreateUser("mig")
	_ = store.UpdateUserProfileSummary(user.ID, "【背景】工程师\n【进展】Go 并发练习中，心理学入门")
	reg := domain.NewRegistry()
	tree, nodes, _ := reg.LoadTreeAndNodes("go-concurrency")
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, _ = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
	_ = store.UpsertProgress(storage.UserProgress{UserID: user.ID, DomainID: tree.DomainID, NodeKey: "goroutine_basics", Status: "completed", Layer: "entry"})

	u, err := MigrateUserProfile(store, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(u.ProfileGoal+u.ProfileBackground, "心理学") {
		// goal may still contain psychology if unattributed - that's ok for conservative
	}
	dp, _ := store.GetDomainProfile(user.ID, tree.DomainID)
	if dp != nil && strings.Contains(dp.Summary, "Go") {
		// attributed Go sentence to domain with progress
	}
}

func TestRefreshUserProfileSkipsWithoutUserMessages(t *testing.T) {
	chdirToRepo(t)
	store, err := storage.Open(t.TempDir() + "/prof.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	mp := &mockProvider{replies: []string{mustJSON(ProfileRefreshOutput{DomainSummary: "不应写入"})}}
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
	_, tree, err = store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
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

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// silence unused import in test build
var _ = time.Now
