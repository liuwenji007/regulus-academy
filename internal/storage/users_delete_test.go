package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDeleteUser_cascadesFKRelatedRows(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "del-user.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("goai")
	if err != nil {
		t.Fatal(err)
	}
	_, tree, err := store.CreateDomainFromTree(
		user.ID, "Go", "go-concurrency", "go", SampleTree("x", "Go"), "{}", DomainSourceSkillPack, false, "",
	)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := store.CreatePlanningSession(user.ID, "intake")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddPlanningMessage(plan.ID, "assistant", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveUserLLMCredentials(user.ID, "openai", "enc", "", "gpt"); err != nil {
		t.Fatal(err)
	}
	if err := store.IncrementLLMMessageCount(user.ID, TodayUTC()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateBindCode(user.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertNodeNote(user.ID, tree.DomainID, "n1", "note"); err != nil {
		t.Fatal(err)
	}
	if err := store.TouchDomainAccess(user.ID, tree.DomainID, "n1"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertDomainProfile(user.ID, tree.DomainID, "摘要", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateDomainBuildJob(user.ID, "Go", "", false); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteUser(user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := store.GetUser(user.ID); err == nil {
		t.Fatal("user should be gone")
	}
	if _, err := store.GetPlanningSession(plan.ID); err == nil {
		t.Fatal("planning session should be gone")
	}
}
