package agent

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestLooksLikeExerciseSubmitPromptVariants(t *testing.T) {
	for _, msg := range []string{
		"题目一\n\n做完后直接把答案发给我。",
		"题目二\n\n做完直接把答案发给我",
	} {
		if !looksLikeExerciseSubmitPrompt(msg) {
			t.Fatalf("应识别作答提示: %q", msg)
		}
	}
	if looksLikeExerciseSubmitPrompt("点击「再来一道」继续练习。") {
		t.Fatal("复习邀请不应识别为作答提示")
	}
}

func TestStripExerciseSubmitSuffix(t *testing.T) {
	got := stripExerciseSubmitSuffix("好的，出一道专项题。\n\n做完直接把答案发给我。")
	if got != "好的，出一道专项题" {
		t.Fatalf("strip=%q", got)
	}
}

func TestExerciseMetaFromContext_omitsEmptyQuestionType(t *testing.T) {
	// 与 adoptPlainTextExercise 一致：无 QT 时不下发，前端走启发式。
	ex := &storage.ExerciseContext{
		Question:     "补全 filter：\n```go\nfunc filter() {}\n```",
		AnswerFormat: "text",
	}
	meta := exerciseMetaFromContext(ex)
	if meta == nil {
		t.Fatal("expected meta")
	}
	if meta.QuestionType != "" {
		t.Fatalf("expected empty questionType, got %q", meta.QuestionType)
	}
	if meta.AnswerFormat != "text" {
		t.Fatalf("answerFormat=%q", meta.AnswerFormat)
	}
}
