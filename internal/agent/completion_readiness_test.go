package agent

import (
	"os"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestLLMCompletionCheckEnabled_defaultOn(t *testing.T) {
	_ = os.Unsetenv("REGULUS_LLM_COMPLETION_CHECK")
	if !LLMCompletionCheckEnabled() {
		t.Fatal("expected enabled by default")
	}
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	if LLMCompletionCheckEnabled() {
		t.Fatal("expected disabled when env=0")
	}
}

func TestMergeCompletionFeedback(t *testing.T) {
	got := mergeCompletionFeedback("很好", "", "可以点亮")
	if got != "很好\n\n可以点亮" {
		t.Fatalf("merge: %q", got)
	}
}

func TestResolveChainReason(t *testing.T) {
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	sctx := &storage.SessionContext{ApplyExercisePassed: false}
	if got := resolveChainReason(DeferNone, sctx, "熟悉"); got != DeferApplyExercise {
		t.Fatalf("apply: %v", got)
	}
	if got := resolveChainReason(DeferConceptCoverage, sctx, "熟悉"); got != DeferConceptCoverage {
		t.Fatalf("coverage: %v", got)
	}
}
