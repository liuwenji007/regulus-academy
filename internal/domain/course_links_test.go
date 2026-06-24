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
	after, afterMod, layer, label := r.resolveDerivationAnchor(parent, "", "go-concurrency", "Go 并发", nil)
	if after != "go_concur_intro" || layer != "intermediate" {
		t.Fatalf("after=%q module=%q layer=%q", after, afterMod, layer)
	}
	if afterMod != "" {
		t.Fatalf("expected topic anchor, got module=%q", afterMod)
	}
	if label != "深入学习 Go 并发" {
		t.Fatalf("label=%q", label)
	}
}

func TestResolveDerivationAnchorModule(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	parent := &storage.KnowledgeTree{
		Modules: []storage.TreeModule{
			{Key: "basics", Label: "基础语法", Nodes: []string{"go_basics"}},
			{Key: "concurrency", Label: "并发编程", Nodes: []string{"go_concur_intro"}},
		},
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{
				{Key: "go_basics", Title: "Go 基础语法"},
			}},
			{Key: "intermediate", Nodes: []storage.TreeNode{
				{Key: "go_concur_intro", Title: "并发入门简介"},
			}},
		},
	}
	after, afterMod, layer, _ := r.resolveDerivationAnchor(parent, "", "go-concurrency", "Go 并发", nil)
	if afterMod != "concurrency" {
		t.Fatalf("module=%q want concurrency", afterMod)
	}
	if after != "" {
		t.Fatalf("topic=%q want empty when module matches", after)
	}
	if layer != "intermediate" {
		t.Fatalf("layer=%q", layer)
	}
}

func TestFindParentDomainSummary(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "d-go", Name: "Go 语言", Slug: "go-language"},
	}
	p := FindParentDomainSummary(all, "go", "")
	if p == nil || p.ID != "d-go" {
		t.Fatalf("got %+v", p)
	}
}

func TestFindParentDomainSummary_subtopicNotParent(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "d-con", Name: "Go 并发", Slug: "go-concurrency", ParentSlug: "go"},
	}
	p := FindParentDomainSummary(all, "go", "d-con")
	if p != nil {
		t.Fatalf("subtopic alone should not match as parent, got %+v", p)
	}
}

func TestFindParentDomainSummary_subtopicWithRealParent(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "d-go", Name: "Go 语言", Slug: "go-language"},
		{ID: "d-con", Name: "Go 并发", Slug: "go-concurrency", ParentSlug: "go"},
	}
	p := FindParentDomainSummary(all, "go", "d-con")
	if p == nil || p.ID != "d-go" {
		t.Fatalf("got %+v want d-go", p)
	}
}

func TestDerivationKeywordsPrefersDB(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	deriv := DerivationResolver(func(domainID string) *DerivationDef {
		if domainID == "child-1" {
			return &DerivationDef{ParentAnchorKeywords: []string{"go mod", "模块"}}
		}
		return nil
	})
	kw := r.derivationKeywords("child-1", "go-modules", "Go 包管理", deriv)
	if len(kw) != 2 || kw[0] != "go mod" {
		t.Fatalf("keywords=%v", kw)
	}
}

func TestResolveCourseLinks_noSelfParent(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	all := []storage.DomainSummary{
		{ID: "d-con", Name: "Go 并发", Slug: "go-concurrency", ParentSlug: "go"},
	}
	links := r.ResolveCourseLinks(all, all[0], nil, nil)
	if links.Parent != nil {
		t.Fatalf("expected no parent banner, got %+v", links.Parent)
	}
}

func TestFindChildDomainSummariesOnlyOnCanonicalParent(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	all := []storage.DomainSummary{
		{ID: "d-go", Name: "Go 语言", Slug: "go-language"},
		{ID: "d-mod", Name: "Go 包管理", Slug: "go-modules"},
		{ID: "d-con", Name: "Go 并发", Slug: "go-concurrency", ParentSlug: "go"},
	}
	onRoot := FindChildDomainSummaries(r, all, all[0])
	if len(onRoot) != 1 || onRoot[0].ID != "d-con" {
		t.Fatalf("on go-language: %+v", onRoot)
	}
	onModules := FindChildDomainSummaries(r, all, all[1])
	if len(onModules) != 0 {
		t.Fatalf("go-modules should not list go-concurrency: %+v", onModules)
	}
}
