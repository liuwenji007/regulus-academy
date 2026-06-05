package storage

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestMigrateProgressByNodeKey(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, oldTree, err := store.CreateDomain("Old Course")
	if err != nil {
		t.Fatal(err)
	}
	_, newTree, err := store.CreateDomain("New Course")
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"a", "b", "c"} {
		if err := store.UpsertProgress(UserProgress{
			UserID: DefaultUserID, DomainID: oldTree.DomainID,
			NodeKey: key, Layer: "entry", Status: "completed", Mastery: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	_ = store.UpsertProgress(UserProgress{
		UserID: DefaultUserID, DomainID: oldTree.DomainID,
		NodeKey: "pending_only", Layer: "entry", Status: "pending",
	})

	valid := map[string]struct{}{"a": {}, "b": {}}
	res, err := store.MigrateProgressByNodeKey(DefaultUserID, oldTree.DomainID, newTree.DomainID, valid)
	if err != nil {
		t.Fatal(err)
	}
	if res.Migrated != 2 || res.Skipped != 1 {
		t.Fatalf("want migrated=2 skipped=1, got %+v", res)
	}

	newList, err := store.ListProgress(DefaultUserID, newTree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(newList) != 2 {
		t.Fatalf("new domain progress len=%d", len(newList))
	}
	oldList, _ := store.ListProgress(DefaultUserID, oldTree.DomainID)
	if len(oldList) != 4 {
		t.Fatalf("old domain progress should remain, len=%d", len(oldList))
	}
}

func TestMigrateProgressByNodeKey_sameDomain(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, tree, _ := store.CreateDomain("X")
	_, err = store.MigrateProgressByNodeKey(DefaultUserID, tree.DomainID, tree.DomainID, map[string]struct{}{"a": {}})
	if err == nil {
		t.Fatal("expected error for same domain")
	}
}

func TestCreateDomainFromTree_forceNew(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	treeJSON, err := SampleTreeJSON("placeholder", "Go Test")
	if err != nil {
		t.Fatal(err)
	}
	var tree KnowledgeTree
	if err := json.Unmarshal([]byte(treeJSON), &tree); err != nil {
		t.Fatal(err)
	}
	slug := "go-force-new-test"

	_, t1, err := store.CreateDomainFromTree(DefaultUserID, "Go 1", slug, &tree, "{}", DomainSourceGenerated, false)
	if err != nil {
		t.Fatal(err)
	}
	tree2 := tree
	_, t2, err := store.CreateDomainFromTree(DefaultUserID, "Go 2", slug, &tree2, "{}", DomainSourceGenerated, false)
	if err != nil {
		t.Fatal(err)
	}
	if t1.DomainID != t2.DomainID {
		t.Fatalf("idempotent should return same id: %s vs %s", t1.DomainID, t2.DomainID)
	}

	if err := store.ClearDomainSlug(DefaultUserID, t1.DomainID); err != nil {
		t.Fatal(err)
	}
	tree3 := tree
	_, t3, err := store.CreateDomainFromTree(DefaultUserID, "Go 3", slug, &tree3, "{}", DomainSourceGenerated, true)
	if err != nil {
		t.Fatal(err)
	}
	if t3.DomainID == t1.DomainID {
		t.Fatal("forceNew should create a new domain id")
	}
}
