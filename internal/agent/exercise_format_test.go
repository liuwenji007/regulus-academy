package agent

import "testing"

func TestNormalizeAnswerFormat(t *testing.T) {
	tests := []struct {
		format, qType, want string
	}{
		{"json", "", "json"},
		{"choice", "", "choice"},
		{"", "code_fill", "json"},
		{"", "bug_find", "json"},
		{"", "short_answer", "text"},
		{"", "unknown", "text"},
	}
	for _, tc := range tests {
		if got := NormalizeAnswerFormat(tc.format, tc.qType); got != tc.want {
			t.Fatalf("NormalizeAnswerFormat(%q,%q)=%q want %q", tc.format, tc.qType, got, tc.want)
		}
	}
}

func TestBuildExerciseContextChoiceFallback(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"仅一个"},
	})
	if ex.AnswerFormat != "text" {
		t.Fatalf("expected text fallback, got %s", ex.AnswerFormat)
	}
	if ex.ChoiceMode != "" {
		t.Fatalf("text fallback should clear choiceMode, got %q", ex.ChoiceMode)
	}
	if len(ex.Choices) != 0 {
		t.Fatal("text fallback should clear choices")
	}
}

func TestBuildExerciseContextChoiceFallback_sparseEmptySlot(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"option1", ""},
	})
	if ex.AnswerFormat != "text" {
		t.Fatalf("expected text fallback for sparse single option, got %s", ex.AnswerFormat)
	}
	if len(ex.Choices) != 0 {
		t.Fatal("text fallback should clear choices")
	}
}

func TestBuildExerciseContextChoice_keepsSparseWithTwoNonEmpty(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"甲", "", "乙"},
	})
	if ex.AnswerFormat != "choice" || len(ex.Choices) != 3 {
		t.Fatalf("want choice with sparse slots, got format=%s choices=%v", ex.AnswerFormat, ex.Choices)
	}
}
