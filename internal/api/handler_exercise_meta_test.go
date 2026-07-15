package api

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestSessionExerciseMeta_prefersActiveExerciseOverLastInReview(t *testing.T) {
	sess := &storage.Session{
		ID:    "s1",
		Phase: "review",
	}
	_ = storage.SaveSessionContext(sess, storage.SessionContext{
		Exercise: &storage.ExerciseContext{
			// 配置类题干可保留 json；空题干的 code_fill 会被 Coerce 成 text。
			AnswerFormat: "json",
			QuestionType: "code_fill",
			Question:     "补全 docker-compose 中 depends_on 与 volumes",
		},
		LastExercise: &storage.ExerciseContext{
			AnswerFormat: "text",
			QuestionType: "short_answer",
		},
	})
	meta := sessionExerciseMeta(sess)
	if meta == nil {
		t.Fatal("expected meta")
	}
	if meta["answerFormat"] != "json" {
		t.Fatalf("should use active Exercise (json config fill), got %v", meta["answerFormat"])
	}
}

func TestSessionExerciseMeta_reviewFallsBackToLastExercise(t *testing.T) {
	sess := &storage.Session{
		ID:    "s1",
		Phase: "review",
	}
	_ = storage.SaveSessionContext(sess, storage.SessionContext{
		LastExercise: &storage.ExerciseContext{
			AnswerFormat: "text",
			QuestionType: "short_answer",
		},
	})
	meta := sessionExerciseMeta(sess)
	if meta == nil || meta["answerFormat"] != "text" {
		t.Fatalf("expected LastExercise meta, got %v", meta)
	}
}

func TestSessionExerciseMeta_exercisePhase(t *testing.T) {
	sess := &storage.Session{
		ID:    "s1",
		Phase: "exercise",
	}
	_ = storage.SaveSessionContext(sess, storage.SessionContext{
		Exercise: &storage.ExerciseContext{
			AnswerFormat: "choice",
			QuestionType: "single_choice",
			Choices:      []string{"A", "B"},
			ChoiceMode:   "single",
		},
	})
	meta := sessionExerciseMeta(sess)
	if meta == nil || meta["answerFormat"] != "choice" {
		t.Fatalf("got %v", meta)
	}
}
