package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestNormalizeToRootTreeGoConcurrency(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	intent := IntentResult{
		Slug:        "go-concurrency",
		DisplayName: "Go 并发",
		Source:      SourceSkillPack,
	}
	out := r.NormalizeToRootTree(intent)
	if out.RootSlug != "go" || out.Slug != "go" {
		t.Fatalf("got slug=%q root=%q", out.Slug, out.RootSlug)
	}
	if out.FocusSlug != "go-concurrency" {
		t.Fatalf("focus=%q", out.FocusSlug)
	}
	if len(out.FocusNodeKeys) < 5 {
		t.Fatalf("focus keys=%v", out.FocusNodeKeys)
	}
	if out.DisplayName != "Go 语言" {
		t.Fatalf("display=%q", out.DisplayName)
	}
}

func TestMergeSkillPackIntoTree(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	pack, packNodes, err := r.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	root := &storage.KnowledgeTree{
		DomainName: "Go 语言",
		Layers: []storage.TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []storage.TreeNode{
				{Key: "go_basics", Title: "Go 基础语法"},
			}},
		},
	}
	nodes := map[string]NodeSpec{
		"go_basics": {Key: "go_basics", Node: "Go 基础语法"},
	}
	focus := MergeSkillPackIntoTree(root, nodes, pack, packNodes)
	if len(focus) < 5 {
		t.Fatalf("focus=%v", focus)
	}
	total := 0
	for _, l := range root.Layers {
		total += len(l.Nodes)
	}
	if total < 6 {
		t.Fatalf("expected merged tree >5 nodes, got %d", total)
	}
	if _, ok := nodes["goroutine_basics"]; !ok {
		t.Fatal("missing skill pack node spec")
	}
}
