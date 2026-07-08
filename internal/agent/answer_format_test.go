package agent

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestValidateExerciseAnswer_json(t *testing.T) {
	ex := &storage.ExerciseContext{AnswerFormat: "json"}
	ok, fb := ValidateExerciseAnswer(ex, "depends_on: db")
	if ok || fb == "" {
		t.Fatalf("invalid json should fail: ok=%v fb=%q", ok, fb)
	}
	ok, _ = ValidateExerciseAnswer(ex, `{"depends_on":["db"]}`)
	if !ok {
		t.Fatal("valid json should pass")
	}
}

func TestValidateExerciseAnswer_text(t *testing.T) {
	ex := &storage.ExerciseContext{AnswerFormat: "text"}
	ok, fb := ValidateExerciseAnswer(ex, "  ")
	if ok {
		t.Fatal("empty text should fail")
	}
	if fb == "" {
		t.Fatal("expected feedback")
	}
}

func TestEnforcePriorExerciseFormat(t *testing.T) {
	prior := &storage.ExerciseContext{
		AnswerFormat: "json",
		QuestionType: "code_fill",
	}
	out := ExerciseOutput{
		Question:     "新题",
		AnswerFormat: "text",
		QuestionType: "short_answer",
		Choices:      []string{"A", "B"},
		ChoiceMode:   "single",
		CorrectChoice: "A",
	}
	EnforcePriorExerciseFormat(prior, &out)
	if out.AnswerFormat != "json" || out.QuestionType != "code_fill" {
		t.Fatalf("format not preserved: %+v", out)
	}
	if len(out.Choices) != 0 || out.CorrectChoice != "" {
		t.Fatalf("choice fields should be cleared: %+v", out)
	}
}
