package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestEvaluateDeferComplete_hybridThreshold(t *testing.T) {
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "1")
	core := []string{"a", "b", "c"}
	deferComplete, reason, uncovered := EvaluateDeferComplete(core, []string{"a"}, nil, "")
	if !deferComplete || reason != DeferConceptCoverage || len(uncovered) != 2 {
		t.Fatalf("want defer with 2 uncovered, got defer=%v reason=%v uncovered=%v", deferComplete, reason, uncovered)
	}
	deferComplete, _, _ = EvaluateDeferComplete(core, []string{"a", "b"}, nil, "")
	if deferComplete {
		t.Fatal("only 1 uncovered should not defer")
	}
	deferComplete, _, _ = EvaluateDeferComplete([]string{"a", "b"}, nil, nil, "")
	if deferComplete {
		t.Fatal("fewer than 3 core should not defer")
	}
}

func TestEvaluateDeferComplete_coverageDisabledByEnv(t *testing.T) {
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "0")
	core := []string{"a", "b", "c", "d"}
	deferComplete, _, _ := EvaluateDeferComplete(core, nil, nil, "")
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

func TestFormatNextExerciseBridge(t *testing.T) {
	if got := FormatNextExerciseBridge(DeferNone, nil); got != "接下来再练一题。" {
		t.Fatalf("empty: %q", got)
	}
	got := FormatNextExerciseBridge(DeferConceptCoverage, []string{"显式思维链", "其他"})
	if !strings.Contains(got, "显式思维链") || !strings.Contains(got, "接下来考查") {
		t.Fatalf("bridge: %q", got)
	}
	if got := FormatNextExerciseBridge(DeferApplyExercise, nil); !strings.Contains(got, "应用级") {
		t.Fatalf("apply bridge: %q", got)
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
	instr := exerciseTaskInstruction(node, nil, nil, false, false)
	if instr == "" || !instrContainsAll(instr, "首题", "choice", "待考查") {
		t.Fatalf("instruction: %s", instr)
	}
	instr2 := exerciseTaskInstruction(node, []string{"a"}, []string{"a"}, false, false)
	if !instrContainsAll(instr2, "第 2 题") {
		t.Fatalf("second: %s", instr2)
	}
	applyInstr := exerciseTaskInstruction(node, []string{"a", "b", "c"}, nil, false, true)
	if !instrContainsAll(applyInstr, "apply", "json", "code_fill", "忽略", "choice") {
		t.Fatalf("apply instruction: %s", applyInstr)
	}
}

func TestEvaluateDeferComplete_applyRequired(t *testing.T) {
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	core := []string{"a", "b"}
	sctx := &storage.SessionContext{}
	deferComplete, reason, _ := EvaluateDeferComplete(core, []string{"a", "b"}, sctx, "熟悉")
	if !deferComplete || reason != DeferApplyExercise {
		t.Fatalf("want apply defer, got defer=%v reason=%v", deferComplete, reason)
	}
	deferComplete, _, _ = EvaluateDeferComplete(core, []string{"a", "b"}, sctx, "入门")
	if deferComplete {
		t.Fatal("entry layer should skip apply gate")
	}
	sctx.ApplyExercisePassed = true
	deferComplete, _, _ = EvaluateDeferComplete(core, []string{"a", "b"}, sctx, "熟悉")
	if deferComplete {
		t.Fatal("apply passed should not defer")
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
