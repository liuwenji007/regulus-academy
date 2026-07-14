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
	NormalizeExerciseContextFormat(ex)
	format := ex.AnswerFormat
	// 仍标为 json 但作答明显是源码：按 text 放行（兼容旧会话）。
	if format == "json" && looksLikeSourceCodeAnswer(answer) && !looksLikeJSONObjectAnswer(answer) {
		format = "text"
		ex.AnswerFormat = "text"
	}
	switch format {
	case "json":
		if answer == "" {
			return false, "请先填写 JSON 答案。"
		}
		if !json.Valid([]byte(answer)) {
			return false, "请使用合法 JSON 作答（对象或数组）。"
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

func looksLikeJSONObjectAnswer(answer string) bool {
	s := strings.TrimSpace(answer)
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func looksLikeSourceCodeAnswer(answer string) bool {
	s := strings.TrimSpace(answer)
	if s == "" {
		return false
	}
	if looksLikeJSONObjectAnswer(s) && json.Valid([]byte(s)) {
		return false
	}
	markers := []string{
		"declare ", "type ", "interface ", "export ", "function ", "const ", "let ", "var ",
		"package ", "func ", ":=", "->", "=>", "class ", "enum ", "#include",
	}
	lower := strings.ToLower(s)
	for _, m := range markers {
		if strings.Contains(lower, strings.ToLower(m)) {
			return true
		}
	}
	return strings.Contains(s, "{") && (strings.Contains(s, ":") || strings.Contains(s, ";"))
}

// EnforcePriorExerciseFormat 非「主动换题」的续练（如 review 薄弱跟题）时，
// 强制沿用上一题的作答方式与题型标签。主动 swap（再来一道 / 二次答错换相似题）不调用本函数，允许按新题选用 text/json。
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
