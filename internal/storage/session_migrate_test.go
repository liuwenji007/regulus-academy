package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrateSessionsByNodeKey(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldTree := &KnowledgeTree{
		Layers: []TreeLayer{{
			Key: "entry", Label: "入门", Nodes: []TreeNode{
				{Key: "old_key", Title: "Goroutine 基础"},
			},
		}},
	}
	newTree := &KnowledgeTree{
		Layers: []TreeLayer{{
			Key: "entry", Label: "入门", Nodes: []TreeNode{
				{Key: "new_key", Title: "Goroutine 基础"},
			},
		}},
	}
	_, oldDom, _ := store.CreateDomain("旧课")
	_, newDom, _ := store.CreateDomain("新课")
	valid := map[string]struct{}{"new_key": {}}

	sess, err := store.CreateSession(DefaultUserID, oldDom.DomainID, "old-slug", "old_key", "completed", nil)
	if err != nil {
		t.Fatal(err)
	}
	sess.Status = "completed"
	_ = store.UpdateSession(sess)
	if _, err := store.AddMessage(sess.ID, "user", "懂了"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage(sess.ID, "assistant", "很好"); err != nil {
		t.Fatal(err)
	}

	res, err := store.MigrateSessionsByNodeKey(DefaultUserID, oldDom.DomainID, newDom.DomainID, "new-slug", valid, oldTree, newTree)
	if err != nil {
		t.Fatal(err)
	}
	if res.Migrated != 1 {
		t.Fatalf("migrated=%d", res.Migrated)
	}

	latest, err := store.FindLatestSession(DefaultUserID, newDom.DomainID, "new_key")
	if err != nil || latest == nil || latest.ID != sess.ID {
		t.Fatalf("session not on new domain: %v", latest)
	}
	if latest.Phase != "completed" {
		t.Fatalf("phase=%s", latest.Phase)
	}
	msgs, err := store.ListMessages(sess.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages=%v err=%v", msgs, err)
	}
}
