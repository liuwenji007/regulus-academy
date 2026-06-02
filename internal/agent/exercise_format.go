package agent

import "github.com/regulus-academy/regulus-academy/internal/storage"

// NormalizeAnswerFormat 将 LLM 输出规范为 text | json | choice
func NormalizeAnswerFormat(format, questionType string) string {
	switch format {
	case "text", "json", "choice":
		return format
	}
	switch questionType {
	case "code_fill", "bug_find":
		return "json"
	case "short_answer":
		return "text"
	default:
		return "text"
	}
}

func normalizeChoiceMode(mode string) string {
	if mode == "multiple" {
		return "multiple"
	}
	return "single"
}

// BuildExerciseContext 从出题 JSON 构建会话内练习上下文
func BuildExerciseContext(out ExerciseOutput) *storage.ExerciseContext {
	format := NormalizeAnswerFormat(out.AnswerFormat, out.QuestionType)
	choices := out.Choices
	if format == "choice" && len(choices) < 2 {
		format = "text"
		choices = nil
	}
	return &storage.ExerciseContext{
		Question:           out.Question,
		QuestionType:       out.QuestionType,
		AnswerFormat:       format,
		Choices:            choices,
		ChoiceMode:         normalizeChoiceMode(out.ChoiceMode),
		ReinforcedConcepts: out.ReinforcedConcepts,
	}
}

func exerciseMetaFromContext(ex *storage.ExerciseContext) *ExerciseMeta {
	if ex == nil {
		return nil
	}
	return &ExerciseMeta{
		AnswerFormat: ex.AnswerFormat,
		Choices:      ex.Choices,
		ChoiceMode:   ex.ChoiceMode,
	}
}
