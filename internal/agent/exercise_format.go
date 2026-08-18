package agent

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// NormalizeAnswerFormat 将 LLM 输出规范为 text | json | choice。
// code_fill / bug_find 默认 text（源码补全）；json 仅表示答案须为合法 JSON/YAML 配置对象。
func NormalizeAnswerFormat(format, questionType string) string {
	switch format {
	case "text", "choice":
		return format
	case "json":
		return "json"
	}
	switch questionType {
	case "code_fill", "bug_find", "short_answer":
		return "text"
	default:
		return "text"
	}
}

// CoerceAnswerFormatForQuestion 仅在 code_fill/bug_find 且题干像配置补全时保留 json。
// 简答即使提到 JSON / docker-compose，答案仍是文字，不弹 JSON 校验框。
func CoerceAnswerFormatForQuestion(format, questionType, question string) string {
	format = NormalizeAnswerFormat(format, questionType)
	if format != "json" {
		return format
	}
	switch strings.TrimSpace(questionType) {
	case "code_fill", "bug_find":
		if questionSuggestsJSONConfigAnswer(question) {
			return "json"
		}
	}
	return "text"
}

// NormalizeExerciseContextFormat 就地纠正已存会话中误标的 answer_format。
func NormalizeExerciseContextFormat(ex *storage.ExerciseContext) {
	if ex == nil {
		return
	}
	ex.AnswerFormat = CoerceAnswerFormatForQuestion(ex.AnswerFormat, ex.QuestionType, ex.Question)
}

func questionSuggestsJSONConfigAnswer(question string) bool {
	q := strings.ToLower(question)
	for _, k := range jsonConfigStrongTokens {
		if strings.Contains(q, k) {
			return true
		}
	}
	fill := strings.Contains(q, "补全") || strings.Contains(q, "填写") ||
		strings.Contains(q, "填入") || strings.Contains(q, "补写")
	lang := strings.Contains(q, "json") || strings.Contains(q, "yaml") || strings.Contains(q, "yml")
	return fill && lang
}

var jsonConfigStrongTokens = []string{
	"docker-compose", "compose.yaml", "compose.yml", "package.json",
	"tsconfig.json", "tsconfig",
	"配置文件", "配置片段", "字段补全", "配置对象",
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
	format := CoerceAnswerFormatForQuestion(out.AnswerFormat, out.QuestionType, out.Question)
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
	NormalizeExerciseContextFormat(ex)
	meta := &ExerciseMeta{
		AnswerFormat: ex.AnswerFormat,
		Choices:      ex.Choices,
	}
	if qt := normalizeQuestionType(ex.QuestionType); qt != "" {
		meta.QuestionType = qt
	}
	if ex.AnswerFormat == "choice" && ex.ChoiceMode != "" {
		meta.ChoiceMode = ex.ChoiceMode
	}
	return meta
}

// normalizeQuestionType 收敛为前端可识别的题型；未知值原样保留（非空时）。
func normalizeQuestionType(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "code_fill", "bug_find", "short_answer":
		return strings.TrimSpace(strings.ToLower(raw))
	default:
		return strings.TrimSpace(raw)
	}
}
