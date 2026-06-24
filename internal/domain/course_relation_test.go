package domain

import (
	"context"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestCandidateParentDomainsPrefersRoots(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "1", Name: "Go 语言", Slug: "go-language", NodeTotal: 12},
		{ID: "2", Name: "Go 并发", Slug: "go-concurrency", ParentSlug: "go", NodeTotal: 8},
		{ID: "3", Name: "Rust", Slug: "rust", NodeTotal: 10},
	}
	intent := IntentResult{Slug: "go-modules", TopicRoot: "go", ScopeBreadth: ScopeNarrow, Source: SourceGenerated}
	cands := CandidateParentDomains(all, intent)
	if len(cands) != 2 {
		t.Fatalf("got %d candidates: %+v", len(cands), cands)
	}
	if cands[0].Slug != "go-language" {
		t.Fatalf("root first, got %+v", cands)
	}
}

func TestCandidateParentDomainsExcludesSelf(t *testing.T) {
	all := []storage.DomainSummary{
		{ID: "1", Name: "Go 包管理", Slug: "go-modules"},
	}
	intent := IntentResult{Slug: "go-modules", TopicRoot: "go"}
	if cands := CandidateParentDomains(all, intent); len(cands) != 0 {
		t.Fatalf("expected none, got %+v", cands)
	}
}

func TestResolveGeneratedCourseRelationAppliesParent(t *testing.T) {
	mock := &mockLLM{jsonReply: `{"parentSlug":"go-language","anchorKeywords":["模块","go mod"],"confidence":0.9,"reason":"子话题"}`}
	intent := IntentResult{
		Slug: "go-modules", DisplayName: "Go 包管理", ScopeBreadth: ScopeNarrow, TopicRoot: "go",
	}
	cands := []storage.DomainSummary{{Slug: "go-language", Name: "Go 语言", NodeTotal: 12}}
	rel, err := ResolveGeneratedCourseRelation(context.Background(), mock, "Go 包管理", intent, cands)
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil || rel.ParentSlug != "go-language" {
		t.Fatalf("got %+v", rel)
	}
	if len(rel.AnchorKeywords) != 2 {
		t.Fatalf("keywords=%v", rel.AnchorKeywords)
	}
}

func TestResolveGeneratedCourseRelationLowConfidenceSkipped(t *testing.T) {
	mock := &mockLLM{jsonReply: `{"parentSlug":"go-language","anchorKeywords":["模块"],"confidence":0.5,"reason":"不确定"}`}
	intent := IntentResult{Slug: "go-modules", DisplayName: "Go 包管理", ScopeBreadth: ScopeNarrow}
	cands := []storage.DomainSummary{{Slug: "go-language", Name: "Go 语言"}}
	rel, err := ResolveGeneratedCourseRelation(context.Background(), mock, "Go 包管理", intent, cands)
	if err != nil {
		t.Fatal(err)
	}
	if rel != nil {
		t.Fatalf("expected nil, got %+v", rel)
	}
}

func TestDerivationFromRelation(t *testing.T) {
	rel := &GeneratedCourseRelation{
		ParentSlug:     "go-language",
		AnchorKeywords: []string{"模块", "go mod"},
	}
	raw := DerivationFromRelation(IntentResult{DisplayName: "Go 包管理"}, rel)
	if raw == "" || !strings.Contains(raw, "parentAnchorKeywords") || !strings.Contains(raw, "深入学习 Go 包管理") {
		t.Fatalf("derivation=%q", raw)
	}
}
