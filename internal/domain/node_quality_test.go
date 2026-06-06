package domain

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestMinExerciseIdeasRequired(t *testing.T) {
	cases := []struct {
		core int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{5, 2},
	}
	for _, c := range cases {
		if got := minExerciseIdeasRequired(c.core); got != c.want {
			t.Fatalf("core=%d got=%d want=%d", c.core, got, c.want)
		}
	}
}

func TestValidateBuildOutput_acceptsInsufficientExerciseIdeas(t *testing.T) {
	var out buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &out); err != nil {
		t.Fatal(err)
	}
	for i := range out.Nodes {
		if out.Nodes[i].Key == "ownership" {
			out.Nodes[i].CoreConcepts = []string{"a", "b"}
			out.Nodes[i].ExerciseIdeas = []string{"only one"}
			break
		}
	}
	tree, nodes, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate})
	if err != nil {
		t.Fatal(err)
	}
	issues := collectTreeQualityIssues(tree, nodes, IntentResult{ScopeBreadth: ScopeModerate})
	hasIdeasIssue := false
	for _, issue := range issues {
		if strings.Contains(issue, "ownership") && strings.Contains(issue, "exercise_ideas") {
			hasIdeasIssue = true
			break
		}
	}
	if !hasIdeasIssue {
		t.Fatalf("expected soft exercise_ideas issue, got %v", issues)
	}
}

func TestCollectTreeQualityIssues_nodeCountSoft(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []storage.TreeNode{{Key: "a", Title: "A"}}},
			{Key: "intermediate", Label: "熟悉", Nodes: []storage.TreeNode{{Key: "b", Title: "B"}}},
			{Key: "advanced", Label: "精通", Nodes: []storage.TreeNode{{Key: "c", Title: "C"}}},
		},
	}
	nodes := map[string]NodeSpec{
		"a": {Key: "a", CoreConcepts: []string{"c"}, ExerciseIdeas: []string{"e"}},
		"b": {Key: "b", CoreConcepts: []string{"c"}, ExerciseIdeas: []string{"e"}},
		"c": {Key: "c", CoreConcepts: []string{"c"}, ExerciseIdeas: []string{"e"}},
	}
	issues := collectTreeQualityIssues(tree, nodes, IntentResult{ScopeBreadth: ScopeBroad})
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "节点总数") && strings.Contains(issue, "建议") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected node count soft issue, got %v", issues)
	}
}
