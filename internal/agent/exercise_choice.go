package agent

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// letteredChoiceLine 匹配题干中的「A. 选项」行（中英文标点）
var letteredChoiceLine = regexp.MustCompile(`(?m)^[ \t]*([A-Da-d])[\.、．\)\:]?[ \t]+(.+?)[ \t]*$`)

// ParseLetteredChoices 从题干文本中提取 A–D 选项；返回去掉选项行后的题干。
// choices[i] 对应字母 A+i 的文案（与 ExpandChoiceAnswer / 选项对照表一致），不随题干出现顺序变化。
func ParseLetteredChoices(question string) (stem string, choices []string, ok bool) {
	matches := letteredChoiceLine.FindAllStringSubmatch(question, -1)
	if len(matches) < 2 {
		return question, nil, false
	}
	byLetter := make(map[byte]string)
	maxIdx := -1
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		letter := strings.ToUpper(strings.TrimSpace(m[1]))
		if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'D' {
			continue
		}
		text := strings.TrimSpace(m[2])
		if text == "" {
			continue
		}
		if _, dup := byLetter[letter[0]]; dup {
			continue
		}
		idx := int(letter[0] - 'A')
		byLetter[letter[0]] = text
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if len(byLetter) < 2 {
		return question, nil, false
	}
	choices = make([]string, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		if text, has := byLetter[byte('A'+i)]; has {
			choices[i] = text
		}
	}
	stem = letteredChoiceLine.ReplaceAllString(question, "")
	stem = strings.TrimSpace(stem)
	if stem == "" {
		stem = strings.TrimSpace(matches[0][0])
	}
	return stem, choices, true
}

// CoerceExerciseOutput 若 LLM 把选项写在题干里但未填 choices，自动转为 choice 题型。
func CoerceExerciseOutput(out *ExerciseOutput) {
	if out == nil {
		return
	}
	format := NormalizeAnswerFormat(out.AnswerFormat, out.QuestionType)
	if format == "choice" && len(out.Choices) >= 2 {
		out.AnswerFormat = "choice"
		return
	}
	stem, choices, ok := ParseLetteredChoices(out.Question)
	if !ok {
		return
	}
	out.Question = stem
	out.Choices = choices
	out.AnswerFormat = "choice"
	if out.ChoiceMode == "" {
		out.ChoiceMode = "single"
	}
}

// ExpandChoiceAnswer 将用户提交的「B」/「选 B」等规范为带完整选项文案，避免批改时字母对错号。
func ExpandChoiceAnswer(ex *storage.ExerciseContext, userMsg string) string {
	if ex == nil || ex.AnswerFormat != "choice" || len(ex.Choices) == 0 {
		return userMsg
	}
	s := strings.TrimSpace(userMsg)
	if s == "" {
		return userMsg
	}
	// 已是「我选择：A. xxx」格式则不再处理
	if strings.Contains(s, "我选择") {
		return userMsg
	}
	letters := extractChoiceLetters(s)
	if len(letters) == 0 {
		return userMsg
	}
	if ex.ChoiceMode != "multiple" && len(letters) > 1 {
		letters = letters[:1]
	}
	var parts []string
	for _, L := range letters {
		idx := int(L - 'A')
		if idx < 0 || idx >= len(ex.Choices) {
			continue
		}
		text := strings.TrimSpace(ex.Choices[idx])
		if text == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%c. %s", L, text))
	}
	if len(parts) == 0 {
		return userMsg
	}
	sep := "；"
	if ex.ChoiceMode != "multiple" {
		sep = ""
	}
	return "我选择：" + strings.Join(parts, sep)
}

func extractChoiceLetters(s string) []rune {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "选")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "。")
	s = strings.TrimSuffix(s, ".")
	// 纯字母或「B项」「选项B」
	var letters []rune
	for _, r := range s {
		if r >= 'a' && r <= 'd' {
			letters = append(letters, unicode.ToUpper(r))
		} else if r >= 'A' && r <= 'D' {
			letters = append(letters, r)
		}
	}
	if len(letters) > 0 {
		return letters
	}
	// 单字符
	if len([]rune(s)) == 1 {
		r := []rune(s)[0]
		if r >= 'A' && r <= 'D' {
			return []rune{r}
		}
		if r >= 'a' && r <= 'd' {
			return []rune{unicode.ToUpper(r)}
		}
	}
	return nil
}

func formatChoicesForPrompt(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【选项对照表】（批改时字母必须与下表一致，禁止凭记忆编造）\n")
	for i, c := range choices {
		if strings.TrimSpace(c) == "" {
			continue
		}
		fmt.Fprintf(&b, "%c. %s\n", 'A'+i, c)
	}
	return strings.TrimRight(b.String(), "\n")
}
