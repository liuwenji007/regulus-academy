package domain

import (
	"encoding/json"
	"testing"
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

func TestValidateBuildOutputRejectsInsufficientExerciseIdeas(t *testing.T) {
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
	_, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate})
	if err == nil {
		t.Fatal("应拒绝 exercise_ideas 不足")
	}
}
