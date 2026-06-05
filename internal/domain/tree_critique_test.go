package domain

import (
	"os"
	"testing"
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
