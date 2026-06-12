package agent

import (
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// InferExerciseLevel 根据作答形式推断题目难度层级。
func InferExerciseLevel(format, questionType string) string {
	format = NormalizeAnswerFormat(format, questionType)
	switch format {
	case "json":
		return domain.ExerciseLevelApply
	case "choice":
		return domain.ExerciseLevelRecognition
	default:
		switch questionType {
		case "code_fill", "bug_find":
			return domain.ExerciseLevelApply
		default:
			return domain.ExerciseLevelRecall
		}
	}
}

// IsApplyExercise 是否为应用级练习（代码/找 bug 等）。
func IsApplyExercise(ex *storage.ExerciseContext) bool {
	if ex == nil {
		return false
	}
	if ex.ExerciseLevel != "" {
		return ex.ExerciseLevel == domain.ExerciseLevelApply
	}
	return InferExerciseLevel(ex.AnswerFormat, ex.QuestionType) == domain.ExerciseLevelApply
}
