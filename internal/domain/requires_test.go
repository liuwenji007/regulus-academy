package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestUnmetRequires(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{
				{Key: "a", Title: "A"},
				{Key: "b", Title: "B", Requires: []string{"a"}},
			}},
		},
	}
	progress := []storage.UserProgress{{NodeKey: "a", Status: "completed"}}
	if got := UnmetRequires(tree, "b", progress); len(got) != 0 {
		t.Fatalf("expected no unmet, got %v", got)
	}
	progress = nil
	if got := UnmetRequires(tree, "b", progress); len(got) != 1 || got[0] != "a" {
		t.Fatalf("expected [a], got %v", got)
	}
}

func TestMergeNodeRequires(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{{Key: "b", Title: "B"}}},
		},
	}
	nodes := map[string]NodeSpec{
		"b": {Key: "b", Requires: []string{"a"}},
	}
	MergeNodeRequires(tree, nodes)
	if len(tree.Layers[0].Nodes[0].Requires) != 1 || tree.Layers[0].Nodes[0].Requires[0] != "a" {
		t.Fatalf("requires not merged: %+v", tree.Layers[0].Nodes[0].Requires)
	}
}
