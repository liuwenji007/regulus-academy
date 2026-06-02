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
}
