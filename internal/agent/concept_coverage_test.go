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
	instr := exerciseTaskInstruction(node, nil, nil, false, false, false, false, "")
	if instr == "" || !instrContainsAll(instr, "首题", "choice", "待考查") {
		t.Fatalf("instruction: %s", instr)
	}
	instr2 := exerciseTaskInstruction(node, []string{"a"}, []string{"a"}, false, false, false, false, "")
	if !instrContainsAll(instr2, "第 2 题") {
		t.Fatalf("second: %s", instr2)
	}
	applyInstr := exerciseTaskInstruction(node, []string{"a", "b", "c"}, nil, false, true, false, false, "")
	if !instrContainsAll(applyInstr, "apply", "code_fill", "json", "text", "忽略", "choice") {
		t.Fatalf("apply instruction: %s", applyInstr)
	}
	swapInstr := exerciseTaskInstruction(node, []string{"a"}, nil, true, false, true, false, "")
	if !instrContainsAll(swapInstr, "勿照搬题干", "可与上一题相同", "命令", "text") {
		t.Fatalf("swap with prior: %s", swapInstr)
	}
	swapNoPrior := exerciseTaskInstruction(node, []string{"a"}, nil, true, false, false, false, "")
	if !instrContainsAll(swapNoPrior, "勿照搬上一题题干", "可更换问法") {
		t.Fatalf("swap without prior: %s", swapNoPrior)
	}
	swapNoPriorEmpty := exerciseTaskInstruction(nil, nil, nil, true, false, false, false, "")
	if !instrContainsAll(swapNoPriorEmpty, "勿照搬上一题题干", "可更换问法") {
		t.Fatalf("swap without prior (no concepts): %s", swapNoPriorEmpty)
	}
	weakInstr := exerciseTaskInstruction(node, []string{"a"}, nil, false, false, true, true, "")
	if !instrContainsAll(weakInstr, "薄弱", "勿照搬题干", "相同薄弱点", "answer_format") {
		t.Fatalf("follow-up weak: %s", weakInstr)
	}
	targetInstr := exerciseTaskInstruction(node, []string{"a"}, nil, false, false, false, false, "鲁棒性")
	if !instrContainsAll(targetInstr, "必须考查", "鲁棒性", "reinforced_concepts") {
		t.Fatalf("target concept: %s", targetInstr)
	}
}

func TestPickNextExerciseTarget_alignsWithBridge(t *testing.T) {
	uncovered := []string{"鲁棒性", "成功率"}
	target := pickNextExerciseTarget(DeferConceptCoverage, uncovered)
	bridge := FormatNextExerciseBridge(DeferConceptCoverage, uncovered)
	if target != "鲁棒性" || !strings.Contains(bridge, "鲁棒性") {
		t.Fatalf("target=%q bridge=%q", target, bridge)
	}
	if pickNextExerciseTarget(DeferApplyExercise, uncovered) != "" {
		t.Fatal("apply defer should not pick concept target")
	}
}

func TestPriorExerciseContext(t *testing.T) {
	current := &storage.ExerciseContext{Question: "当前题"}
	last := &storage.ExerciseContext{Question: "上一题"}
	sctx := &storage.SessionContext{Exercise: current, LastExercise: last, RecentMistakes: []string{"a"}}

	got, weak := priorExerciseContext(sctx, "exercise", true)
	if got != current || weak {
		t.Fatalf("swap in exercise: got=%v weak=%v", got, weak)
	}
	got, weak = priorExerciseContext(sctx, "review", false)
	if got != last || !weak {
		t.Fatalf("review follow-up: got=%v weak=%v", got, weak)
	}
	sctxReview := &storage.SessionContext{LastExercise: last, RecentMistakes: []string{"a"}}
	got, weak = priorExerciseContext(sctxReview, "review", true)
	if got != last || !weak {
		t.Fatalf("review swap: got=%v weak=%v", got, weak)
	}
	sctxNoMistake := &storage.SessionContext{LastExercise: last}
	got, weak = priorExerciseContext(sctxNoMistake, "review", true)
	if got != last || weak {
		t.Fatalf("review swap without mistakes: got=%v weak=%v", got, weak)
	}
	got, weak = priorExerciseContext(&storage.SessionContext{}, "explain", true)
	if got != nil || weak {
		t.Fatalf("no prior: got=%v weak=%v", got, weak)
	}
}

func TestBuildContext_TaskExerciseIncludesPriorExercise(t *testing.T) {
	in := sampleInput()
	in.PriorExercise = &storage.ExerciseContext{
		Question:           "channel 同步握手",
		AnswerFormat:       "choice",
		QuestionType:       "short_answer",
		ReinforcedConcepts: []string{"无缓冲 channel"},
	}
	ctx := buildContext(in, TaskExercise)
	if !strings.Contains(ctx, "【上一题（勿照搬题干）】") || !strings.Contains(ctx, "channel 同步握手") {
		t.Fatalf("context: %s", ctx)
	}
}

func TestBuildContext_TaskExerciseIncludesExerciseTarget(t *testing.T) {
	in := sampleInput()
	in.ExerciseTarget = "鲁棒性"
	ctx := buildContext(in, TaskExercise)
	if !strings.Contains(ctx, "【本题考查】鲁棒性") {
		t.Fatalf("context: %s", ctx)
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
