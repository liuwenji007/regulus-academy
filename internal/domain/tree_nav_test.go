package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestNextNodeAfter(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{
				{Key: "a", Title: "A"},
				{Key: "b", Title: "B"},
			}},
			{Key: "intermediate", Nodes: []storage.TreeNode{
				{Key: "c", Title: "C"},
			}},
		},
	}
	key, layer, title, ok := NextNodeAfter(tree, "a")
	if !ok || key != "b" || layer != "entry" || title != "B" {
		t.Fatalf("after a: key=%s layer=%s title=%s ok=%v", key, layer, title, ok)
	}
	key, layer, _, ok = NextNodeAfter(tree, "b")
	if !ok || key != "c" || layer != "intermediate" {
		t.Fatalf("after b: key=%s layer=%s ok=%v", key, layer, ok)
	}
	if _, _, _, ok := NextNodeAfter(tree, "c"); ok {
		t.Fatal("c should have no next")
	}
}

func TestNextUncompletedNodeAfterSkipsCompleted(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{
				{Key: "a", Title: "A"},
				{Key: "b", Title: "B"},
				{Key: "c", Title: "C"},
			}},
		},
	}
	completed := map[string]bool{"b": true, "c": true}
	key, _, title, ok := NextUncompletedNodeAfter(tree, "a", completed)
	if ok {
		t.Fatalf("a 之后仅剩已完成节点，应无下一节: key=%s title=%s", key, title)
	}
	completed = map[string]bool{"b": true}
	key, _, title, ok = NextUncompletedNodeAfter(tree, "a", completed)
	if !ok || key != "c" || title != "C" {
		t.Fatalf("应跳过已完成的 b 到 c: key=%s title=%s ok=%v", key, title, ok)
	}
}
