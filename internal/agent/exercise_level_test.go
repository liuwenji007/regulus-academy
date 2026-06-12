package agent

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestInferExerciseLevel(t *testing.T) {
	if got := InferExerciseLevel("choice", "short_answer"); got != domain.ExerciseLevelRecognition {
		t.Fatalf("choice: %s", got)
	}
	if got := InferExerciseLevel("json", "code_fill"); got != domain.ExerciseLevelApply {
		t.Fatalf("json: %s", got)
	}
	if got := InferExerciseLevel("text", "short_answer"); got != domain.ExerciseLevelRecall {
		t.Fatalf("text: %s", got)
	}
}

func TestIsApplyExercise(t *testing.T) {
	if !IsApplyExercise(&storage.ExerciseContext{AnswerFormat: "json", QuestionType: "code_fill"}) {
		t.Fatal("json code_fill should be apply")
	}
	if IsApplyExercise(&storage.ExerciseContext{AnswerFormat: "choice", QuestionType: "short_answer"}) {
		t.Fatal("choice should not be apply")
	}
}
