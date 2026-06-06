package storage

import (
	"path/filepath"
	"testing"
)

func TestUpdateDomainTreeInPlacePreservesProgress(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tree := &KnowledgeTree{
		DomainName: "Test",
		Layers: []TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []TreeNode{{Key: "old_key", Title: "旧节点"}}},
			{Key: "intermediate", Label: "熟悉", Nodes: []TreeNode{}},
			{Key: "advanced", Label: "精通", Nodes: []TreeNode{}},
		},
	}
	_, saved, err := store.CreateDomainFromTree(DefaultUserID, "Test", "test-slug", tree, "{}", DomainSourceGenerated, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertProgress(UserProgress{
		UserID: DefaultUserID, DomainID: saved.DomainID, NodeKey: "old_key",
		Layer: "entry", Status: "completed", Mastery: 1,
	}); err != nil {
		t.Fatal(err)
	}

	saved.Layers[0].Nodes = append(saved.Layers[0].Nodes, TreeNode{Key: "new_key", Title: "新节点"})
	newVersion, err := store.UpdateDomainTreeInPlace(DefaultUserID, saved.DomainID, saved, `{"new_key":{"key":"new_key","core_concepts":["x"]}}`, []string{"new_key"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if newVersion != 2 {
		t.Fatalf("version=%d", newVersion)
	}

	progress, err := store.ListProgress(DefaultUserID, saved.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 1 || progress[0].NodeKey != "old_key" || progress[0].Status != "completed" {
		t.Fatalf("progress=%+v", progress)
	}
}
