package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestShouldDeferComplete_hybridThreshold(t *testing.T) {
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "1")
	core := []string{"a", "b", "c"}
	deferComplete, uncovered := ShouldDeferComplete(core, []string{"a"})
	if !deferComplete || len(uncovered) != 2 {
		t.Fatalf("want defer with 2 uncovered, got defer=%v uncovered=%v", deferComplete, uncovered)
	}
	deferComplete, _ = ShouldDeferComplete(core, []string{"a", "b"})
	if deferComplete {
		t.Fatal("only 1 uncovered should not defer")
	}
	deferComplete, _ = ShouldDeferComplete([]string{"a", "b"}, nil)
	if deferComplete {
		t.Fatal("fewer than 3 core should not defer")
	}
}

func TestShouldDeferComplete_disabledByEnv(t *testing.T) {
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	core := []string{"a", "b", "c", "d"}
	deferComplete, _ := ShouldDeferComplete(core, nil)
	if deferComplete {
		t.Fatal("coverage gate should be off")
	}
}

func TestMergeExplainedConcepts_dedup(t *testing.T) {
	sctx := &storage.SessionContext{}
	core := []string{"goroutine 轻量", "go 启动"}
	MergeExplainedConcepts(sctx, core, []string{"goroutine 轻量"})
	MergeExplainedConcepts(sctx, core, []string{"轻量"})
	if len(sctx.ExplainedConcepts) != 1 {
		t.Fatalf("expected 1 explained, got %v", sctx.ExplainedConcepts)
	}
}

func TestEnsureExplainedConcepts_legacySession(t *testing.T) {
	sctx := &storage.SessionContext{TestedConcepts: []string{"a"}}
	core := []string{"a", "b"}
	EnsureExplainedConcepts(sctx, core)
	if len(sctx.ExplainedConcepts) != 1 || sctx.ExplainedConcepts[0] != "a" {
		t.Fatalf("legacy: %v", sctx.ExplainedConcepts)
	}
}

func TestNextConceptToDeepen(t *testing.T) {
	core := []string{"a", "b", "c"}
	if got := NextConceptToDeepen(core, nil, nil, false); got != "a" {
		t.Fatalf("pre-exercise: %q", got)
	}
	if got := NextConceptToDeepen(core, []string{"a"}, []string{"a"}, true); got != "b" {
		t.Fatalf("after pass: %q", got)
	}
}

func TestNextExerciseTargetConcept(t *testing.T) {
	core := []string{"a", "b"}
	if got := NextExerciseTargetConcept(core, nil); got != "a" {
		t.Fatalf("first: %q", got)
	}
	if got := NextExerciseTargetConcept(core, []string{"a"}); got != "b" {
		t.Fatalf("second: %q", got)
	}
}

func TestMergeTestedConcepts_normalizesToCore(t *testing.T) {
	core := []string{"goroutine 是轻量级执行单元", "go 关键字启动"}
	got := MergeTestedConcepts(nil, core, []string{"轻量级执行单元"})
	if len(got) != 1 || got[0] != core[0] {
		t.Fatalf("merge: %v", got)
	}
}

func TestExerciseTaskInstruction_firstAndSecond(t *testing.T) {
	node := &domain.NodeSpec{
		CoreConcepts:       []string{"a", "b", "c"},
		FirstExerciseLevel: "recognition",
	}
	instr := exerciseTaskInstruction(node, nil, nil, false)
	if instr == "" || !instrContainsAll(instr, "首题", "choice", "待考查") {
		t.Fatalf("instruction: %s", instr)
	}
	instr2 := exerciseTaskInstruction(node, []string{"a"}, []string{"a"}, false)
	if !instrContainsAll(instr2, "第 2 题") {
		t.Fatalf("second: %s", instr2)
	}
}

func instrContainsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestStrictConceptCoverageEnabled_defaultOn(t *testing.T) {
	_ = os.Unsetenv("REGULUS_STRICT_CONCEPT_COVERAGE")
	if !StrictConceptCoverageEnabled() {
		t.Fatal("expected enabled by default")
	}
}
