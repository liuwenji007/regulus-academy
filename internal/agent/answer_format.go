package agent

import (
	"encoding/json"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// ValidateExerciseAnswer 作答格式校验；不通过时返回给用户看的反馈（保持当前题不变）。
func ValidateExerciseAnswer(ex *storage.ExerciseContext, answer string) (ok bool, feedback string) {
	if ex == nil {
		return true, ""
	}
	answer = strings.TrimSpace(answer)
	switch ex.AnswerFormat {
	case "json":
		if answer == "" {
			return false, "请先填写 JSON 或代码答案。"
		}
		if !json.Valid([]byte(answer)) {
			return false, "你没有按要求使用 JSON 格式作答。请修正格式后重新提交"
		}
	case "choice":
		return true, ""
	default:
		if answer == "" {
			return false, "请先写下你的答案。"
		}
	}
	return true, ""
}

// EnforcePriorExerciseFormat 换题时强制沿用上一题的作答方式与题型标签。
func EnforcePriorExerciseFormat(prior *storage.ExerciseContext, out *ExerciseOutput) {
	if prior == nil || out == nil {
		return
	}
	if prior.AnswerFormat != "" {
		out.AnswerFormat = prior.AnswerFormat
	}
	if prior.QuestionType != "" {
		out.QuestionType = prior.QuestionType
	}
	if prior.AnswerFormat != "choice" {
		out.Choices = nil
		out.ChoiceMode = ""
		out.CorrectChoice = ""
		out.CorrectChoices = nil
	}
}
