package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestCollectStructuredFindings_missingBeats(t *testing.T) {
	tree := &storage.KnowledgeTree{
		DomainName: "Test",
		Layers: []storage.TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []storage.TreeNode{{Key: "a", Title: "A"}}},
			{Key: "intermediate", Label: "熟悉", Nodes: []storage.TreeNode{{Key: "b", Title: "B"}}},
			{Key: "advanced", Label: "精通", Nodes: []storage.TreeNode{{Key: "c", Title: "C"}}},
		},
	}
	nodes := map[string]NodeSpec{
		"a": {
			Key: "a", Node: "A",
			CoreConcepts:   []string{"concept a"},
			Boundaries:     []string{"b"},
			CommonMistakes: []string{"m"},
			ExerciseIdeas:  []string{"e"},
		},
		"b": {Key: "b", Node: "B", CoreConcepts: []string{"c"}, Boundaries: []string{"x"}, CommonMistakes: []string{"m"}, ExerciseIdeas: []string{"e"}, TeachingBeats: []ConceptBeat{{Concept: "c", MustTeach: []string{"a", "b"}}}},
		"c": {Key: "c", Node: "C", CoreConcepts: []string{"c"}, Boundaries: []string{"x"}, CommonMistakes: []string{"m"}, ExerciseIdeas: []string{"e"}, TeachingBeats: []ConceptBeat{{Concept: "c", MustTeach: []string{"a", "b"}}}},
	}
	findings := collectStructuredFindings(tree, nodes, IntentResult{ScopeBreadth: ScopeModerate})
	var beatsMissing bool
	for _, f := range findings {
		if f.Code == CodeMissingTeachingBeats && f.NodeKey == "a" {
			beatsMissing = true
			if !f.AutoFixable || f.FixKind != FixKindEnrichNode {
				t.Fatalf("expected autoFixable enrich_node, got %+v", f)
			}
		}
	}
	if !beatsMissing {
		t.Fatal("expected MISSING_TEACHING_BEATS for node a")
	}
}

func TestScoreAuditFindings_grade(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityWarn, Dimension: DimensionTeachingAlignment},
		{Severity: SeverityInfo, Dimension: DimensionStructure},
	}
	summary, dims := scoreAuditFindings(findings)
	if summary.Score != 94 {
		t.Fatalf("score=%d want 94", summary.Score)
	}
	if summary.Grade != "A" {
		t.Fatalf("grade=%s want A", summary.Grade)
	}
	if dims[DimensionTeachingAlignment].Score != 95 {
		t.Fatalf("dim score=%d", dims[DimensionTeachingAlignment].Score)
	}
}

func TestFindingsByIDs(t *testing.T) {
	all := []Finding{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	got := FindingsByIDs(all, []string{"b", "c"})
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestApplyOptimizePatches_fixRequires(t *testing.T) {
	nodes := map[string]NodeSpec{
		"a": {Key: "a", Requires: []string{"missing", "b"}},
		"b": {Key: "b"},
	}
	patches := []OptimizePatchItem{
		{
			ID: "patch:a", NodeKey: "a",
			After: NodePatchFields{Requires: []string{"b"}},
		},
	}
	merged, err := ApplyOptimizePatches(nodes, patches, []string{"patch:a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged["a"].Requires) != 1 || merged["a"].Requires[0] != "b" {
		t.Fatalf("requires=%v", merged["a"].Requires)
	}
}
