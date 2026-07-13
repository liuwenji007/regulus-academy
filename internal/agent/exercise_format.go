package agent

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

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
	CoerceExerciseOutput(&out)
	out.Question = SanitizeExerciseQuestion(out.Question)
	format := NormalizeAnswerFormat(out.AnswerFormat, out.QuestionType)
	choices := out.Choices
	choiceMode := ""
	if format == "choice" && nonEmptyChoiceCount(choices) < 2 {
		format = "text"
		choices = nil
	}
	if format == "choice" {
		choiceMode = normalizeChoiceMode(out.ChoiceMode)
	}
	ex := &storage.ExerciseContext{
		Question:           out.Question,
		QuestionType:       out.QuestionType,
		AnswerFormat:       format,
		Choices:            choices,
		ChoiceMode:         choiceMode,
		ReinforcedConcepts: out.ReinforcedConcepts,
		ExerciseLevel:      InferExerciseLevel(format, out.QuestionType),
	}
	applyCorrectAnswer(out, ex)
	return ex
}

func applyCorrectAnswer(out ExerciseOutput, ex *storage.ExerciseContext) {
	if ex == nil || ex.AnswerFormat != "choice" {
		return
	}
	letters, ok := normalizeCorrectAnswer(out, ex.ChoiceMode)
	if !ok || len(letters) == 0 {
		return
	}
	if ex.ChoiceMode == "multiple" {
		ex.CorrectChoices = runesToLetterStrings(letters)
		return
	}
	ex.CorrectChoice = string(letters[0])
}

// CopyExerciseContext 深拷贝练习上下文，供落库上一题时使用。
func CopyExerciseContext(ex *storage.ExerciseContext) *storage.ExerciseContext {
	if ex == nil {
		return nil
	}
	cp := *ex
	if len(ex.Choices) > 0 {
		cp.Choices = append([]string(nil), ex.Choices...)
	}
	if len(ex.CorrectChoices) > 0 {
		cp.CorrectChoices = append([]string(nil), ex.CorrectChoices...)
	}
	if len(ex.ReinforcedConcepts) > 0 {
		cp.ReinforcedConcepts = append([]string(nil), ex.ReinforcedConcepts...)
	}
	return &cp
}

func formatPriorExerciseForPrompt(ex *storage.ExerciseContext) string {
	if ex == nil || strings.TrimSpace(ex.Question) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("【上一题（勿照搬题干）】\n")
	b.WriteString(strings.TrimSpace(ex.Question))
	b.WriteString("\n")
	if ex.AnswerFormat != "" {
		fmt.Fprintf(&b, "作答方式：%s\n", ex.AnswerFormat)
	}
	if ex.QuestionType != "" {
		fmt.Fprintf(&b, "题型：%s\n", ex.QuestionType)
	}
	if len(ex.ReinforcedConcepts) > 0 {
		fmt.Fprintf(&b, "考查：%s\n", strings.Join(ex.ReinforcedConcepts, "；"))
	}
	return strings.TrimRight(b.String(), "\n")
}

func exerciseMetaFromContext(ex *storage.ExerciseContext) *ExerciseMeta {
	if ex == nil {
		return nil
	}
	meta := &ExerciseMeta{
		AnswerFormat: ex.AnswerFormat,
		Choices:      ex.Choices,
	}
	if ex.AnswerFormat == "choice" && ex.ChoiceMode != "" {
		meta.ChoiceMode = ex.ChoiceMode
	}
	return meta
}
