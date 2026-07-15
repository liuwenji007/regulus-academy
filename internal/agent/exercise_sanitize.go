package agent

import (
	"regexp"
	"strings"
)

var (
	exerciseBacktickTailRe    = `(?:\s*的写法)?[。]?`
	exerciseBacktickExampleRe = regexp.MustCompile(`[，,]?\s*如[：:]?\s*` + "`[^`]*`" + exerciseBacktickTailRe)
	exerciseForExampleRe       = regexp.MustCompile(`[，,]?\s*例如[：:]?\s*` + "`[^`]*`" + exerciseBacktickTailRe)
	exercisePascalExampleRe    = regexp.MustCompile(`[，,]?\s*如[：:]?\s*(?:[A-Z][A-Za-z0-9]*(?:\s*\|\s*'[^']+')?(?:\s+|$)){2,}[A-Za-z0-9|'|\s]+`)
	exerciseExplicitAnswerRe   = regexp.MustCompile(`(?m)^\s*答案[是为：:]\s*.+$`)
)

// SanitizeExerciseQuestion 去掉题干中泄露标准答案的示例说明（LLM 偶发在「如 …」里写出可照抄答案）。
func SanitizeExerciseQuestion(question string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		return q
	}
	for _, re := range []*regexp.Regexp{
		exerciseBacktickExampleRe,
		exerciseForExampleRe,
		exercisePascalExampleRe,
	} {
		q = re.ReplaceAllString(q, "")
	}
	q = exerciseExplicitAnswerRe.ReplaceAllString(q, "")
	q = strings.ReplaceAll(q, "（，", "（")
	q = strings.ReplaceAll(q, "(,", "(")
	q = strings.ReplaceAll(q, "，）", "）")
	q = strings.ReplaceAll(q, ",)", ")")
	q = strings.ReplaceAll(q, "（）", "")
	q = strings.ReplaceAll(q, "()", "")
	return strings.TrimSpace(q)
}
