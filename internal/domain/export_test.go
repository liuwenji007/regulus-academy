package domain

import (
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestExportToFiles(t *testing.T) {
	tree := &storage.KnowledgeTree{
		DomainName: "Go 并发",
		Layers: []storage.TreeLayer{
			{
				Key: "entry", Label: "入门", Time: "约 2 小时", Goal: "看懂并发代码",
				Nodes: []storage.TreeNode{{Key: "waitgroup", Title: "sync.WaitGroup 等待完成"}},
			},
		},
	}
	nodes := map[string]NodeSpec{
		"waitgroup": {
			Key:            "waitgroup",
			Node:           "sync.WaitGroup 等待完成",
			Layer:          "入门",
			CoreConcepts:   []string{"Add 在 Wait 之前"},
			CommonMistakes: []string{"重复 Done"},
			Boundaries:     []string{"不讲 channel"},
			ExerciseIdeas:  []string{"用 WaitGroup 等待 3 个 goroutine"},
		},
	}

	files, err := ExportToFiles(tree, "go-concurrency", "测试描述", 1, nodes)
	if err != nil {
		t.Fatalf("ExportToFiles: %v", err)
	}
	if !strings.Contains(files["tree.yaml"], "go-concurrency") {
		t.Fatalf("tree.yaml missing slug: %s", files["tree.yaml"])
	}
	if !strings.Contains(files["nodes/waitgroup.yaml"], "core_concepts") {
		t.Fatalf("node yaml missing core_concepts: %s", files["nodes/waitgroup.yaml"])
	}
}

func TestSlugifyExportName(t *testing.T) {
	if got := slugifyExportName("Rust 语言"); got != "rust" {
		t.Fatalf("got %q", got)
	}
	if got := slugifyExportName(""); got != "generated-domain" {
		t.Fatalf("got %q", got)
	}
}

func TestListPublicDomains(t *testing.T) {
	r := NewRegistry()
	list, err := r.ListPublicDomains()
	if err != nil {
		t.Fatalf("ListPublicDomains: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least go-concurrency")
	}
	found := false
	for _, d := range list {
		if d.Slug == "go-concurrency" {
			found = true
			if d.NodeCount < 5 {
				t.Fatalf("unexpected node count: %d", d.NodeCount)
			}
			if d.Version < 1 {
				t.Fatalf("unexpected version: %d", d.Version)
			}
		}
	}
	if !found {
		t.Fatal("go-concurrency not in public catalog")
	}
}
