package domain

import (
	"os"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestCollectTreeQualityIssues(t *testing.T) {
	nodes := map[string]NodeSpec{
		"a": {
			Node: "A", CoreConcepts: []string{"c1", "c2"},
			ExerciseIdeas: []string{"e1"},
		},
		"b": {
			Node: "B", CoreConcepts: []string{"c1"},
			Boundaries: []string{"b"}, CommonMistakes: []string{"m"},
			Requires:    []string{"missing"},
		},
	}
	issues := collectTreeQualityIssues(nodes, 2)
	if len(issues) < 3 {
		t.Fatalf("expected multiple issues, got %v", issues)
	}
}

func TestBuildTreeCritiqueUserMessage_includesNodeDetails(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{
				Key: "entry", Label: "入门", Goal: "建立框架",
				Nodes: []storage.TreeNode{{Key: "basics", Title: "基础语法"}},
			},
		},
	}
	nodes := map[string]NodeSpec{
		"basics": {
			Key: "basics", Node: "基础语法", Layer: "入门",
			CoreConcepts:   []string{"变量与类型"},
			Boundaries:     []string{"不讲 OOP"},
			CommonMistakes: []string{"混淆 = 与 :="},
			ExerciseIdeas:  []string{"读一段变量声明"},
		},
	}
	msg := buildTreeCritiqueUserMessage(tree, nodes, nil, IntentResult{
		DisplayName:  "Python",
		ScopeBreadth: ScopeBroad,
	})
	for _, want := range []string{
		"【层内节点顺序】",
		"basics: 基础语法",
		"【节点明细】",
		"### basics: 基础语法",
		"core_concepts: 变量与类型",
		"exercise_ideas（1 条，至少 1）",
		"boundaries: 不讲 OOP",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in critique prompt:\n%s", want, msg)
		}
	}
}

func TestTreeCritiqueEnabled_defaultOn(t *testing.T) {
	_ = os.Unsetenv("REGULUS_TREE_CRITIQUE")
	if !TreeCritiqueEnabled() {
		t.Fatal("expected enabled by default")
	}
	t.Setenv("REGULUS_TREE_CRITIQUE", "0")
	if TreeCritiqueEnabled() {
		t.Fatal("expected disabled")
	}
}
