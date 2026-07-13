package storage

import (
	"strings"
	"testing"
	"time"
)

func TestParseBackgroundGoal_stripsProgress(t *testing.T) {
	in := "【背景】前端开发\n【进展】心理学已学人本主义，Go 并发进行中"
	got := ParseBackgroundGoal(in)
	if containsStr(got, "心理学") || containsStr(got, "【进展】") {
		t.Fatalf("should strip progress: %q", got)
	}
	if !containsStr(got, "前端") {
		t.Fatalf("should keep background: %q", got)
	}
}

func TestComposeForCoach_excludesOtherCourseProgress(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpdateUserProfileSummary(user.ID, "【背景】工程师\n【进展】心理学人本主义已掌握")
	_, tree, err := store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", SampleTree("x", "Go"), "{}", DomainSourceSkillPack, false, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpsertDomainProfile(user.ID, tree.DomainID, "理解 goroutine 与 channel 基础", time.Now().UTC())

	got, err := store.ComposeForCoach(user.ID, tree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	if containsStr(got, "心理学") {
		t.Fatalf("coach compose leaked other course: %q", got)
	}
	if !containsStr(got, "goroutine") {
		t.Fatalf("missing domain summary: %q", got)
	}
}

func TestUpsertDomainProfile_outOfOrder(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir + "/ood.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, _ := store.CreateUser("测试")
	_, tree, _ := store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", SampleTree("x", "Go"), "{}", DomainSourceSkillPack, false, "")

	newer := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	older := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	if err := store.UpsertDomainProfile(user.ID, tree.DomainID, "新摘要", newer); err != nil {
		t.Fatal(err)
	}
	_ = store.UpsertDomainProfile(user.ID, tree.DomainID, "旧摘要", older)

	dp, err := store.GetDomainProfile(user.ID, tree.DomainID)
	if err != nil || dp == nil {
		t.Fatal(err)
	}
	if dp.Summary != "新摘要" {
		t.Fatalf("out-of-order overwrite: %q", dp.Summary)
	}
}

func TestDeleteDomain_cascadesDomainProfile(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir + "/del.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, _ := store.CreateUser("测试")
	_, tree, _ := store.CreateDomainFromTree(user.ID, "Go", "go-concurrency", "go", SampleTree("x", "Go"), "{}", DomainSourceSkillPack, false, "")
	_ = store.UpsertDomainProfile(user.ID, tree.DomainID, "摘要", time.Now().UTC())
	if err := store.DeleteDomain(user.ID, tree.DomainID); err != nil {
		t.Fatal(err)
	}
	dp, _ := store.GetDomainProfile(user.ID, tree.DomainID)
	if dp != nil {
		t.Fatal("domain profile should be deleted")
	}
}

func TestComposeLegacySummary(t *testing.T) {
	got := ComposeLegacySummary("前端", "学 Go", "偏实战")
	if !containsStr(got, "【背景】") || !containsStr(got, "【目标】") {
		t.Fatalf("legacy=%q", got)
	}
	if containsStr(got, "【进展】") {
		t.Fatal("legacy should use 目标 not 进展")
	}
}

func TestListUsers_withDomainProfiles_noDeadlock(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir + "/listusers.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	u1, err := store.CreateUser("角色甲")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := store.CreateUser("角色乙")
	if err != nil {
		t.Fatal(err)
	}
	_, tree, err := store.CreateDomainFromTree(u1.ID, "Go", "go-concurrency", "go", SampleTree("x", "Go"), "{}", DomainSourceSkillPack, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertDomainProfile(u1.ID, tree.DomainID, "理解 channel", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- func() error { _, err := store.ListUsers(); return err }() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ListUsers deadlocked with open user rows + nested query")
	}

	list, err := store.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("users=%d", len(list))
	}
	var found bool
	for _, u := range list {
		if u.ID == u2.ID {
			continue
		}
		if u.ID == u1.ID && len(u.DomainProfiles) == 1 && u.DomainProfiles[0].Summary == "理解 channel" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected domain profiles on list users")
	}
}

func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}
