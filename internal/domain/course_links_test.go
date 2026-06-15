package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestEnrichTopicMetaGoConcurrency(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	intent := IntentResult{
		Slug:        "go-concurrency",
		DisplayName: "Go 并发",
		Source:      SourceSkillPack,
	}
	out := r.EnrichTopicMeta(intent)
	if out.Slug != "go-concurrency" {
		t.Fatalf("slug=%q", out.Slug)
	}
	if out.ParentSlug != "go" || !out.IsSubtopic {
		t.Fatalf("parent=%q isSub=%v", out.ParentSlug, out.IsSubtopic)
	}
}

func TestEnrichTopicMetaGeneratedNoParent(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	intent := IntentResult{
		Slug:        "go-concurrency",
		DisplayName: "Go 并发",
		Source:      SourceGenerated,
	}
	out := r.EnrichTopicMeta(intent)
	if out.Slug != "go-concurrency" {
		t.Fatalf("slug=%q", out.Slug)
	}
	if out.ParentSlug != "" || out.IsSubtopic {
		t.Fatalf("generated should stay independent: parent=%q isSub=%v", out.ParentSlug, out.IsSubtopic)
	}
	if out.TopicRoot != "go" {
		t.Fatalf("topicRoot=%q", out.TopicRoot)
	}
}

func TestResolveDerivationAnchor(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	parent := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{
				{Key: "go_basics", Title: "Go 基础语法"},
			}},
			{Key: "intermediate", Nodes: []storage.TreeNode{
				{Key: "go_funcs", Title: "函数与方法"},
				{Key: "go_concur_intro", Title: "并发入门简介"},
			}},
		},
	}
	after, layer, label := r.resolveDerivationAnchor(parent, "go-concurrency", "Go 并发")
	if after != "go_concur_intro" || layer != "intermediate" {
		t.Fatalf("after=%q layer=%q", after, layer)
	}
	if label != "深入学习 Go 并发" {
		t.Fatalf("label=%q", label)
	}
}

func TestFindParentDomainSummary(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "d-go", Name: "Go 语言", Slug: "go-language"},
	}
	p := FindParentDomainSummary(all, "go")
	if p == nil || p.ID != "d-go" {
		t.Fatalf("got %+v", p)
	}
}
