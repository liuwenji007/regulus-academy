package agent

import (
	"os"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
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

func TestMergeTestedConcepts_normalizesToCore(t *testing.T) {
	core := []string{"goroutine 是轻量级执行单元", "go 关键字启动"}
	got := MergeTestedConcepts(nil, core, []string{"轻量级执行单元"})
	if len(got) != 1 || got[0] != core[0] {
		t.Fatalf("merge: %v", got)
	}
}

func TestExerciseTaskInstruction_uncovered(t *testing.T) {
	node := &domain.NodeSpec{
		CoreConcepts: []string{"a", "b", "c"},
	}
	instr := exerciseTaskInstruction(node, []string{"a"}, false)
	if instr == "" || len(instr) < 10 {
		t.Fatal("expected instruction")
	}
}

func TestStrictConceptCoverageEnabled_defaultOn(t *testing.T) {
	_ = os.Unsetenv("REGULUS_STRICT_CONCEPT_COVERAGE")
	if !StrictConceptCoverageEnabled() {
		t.Fatal("expected enabled by default")
	}
}
