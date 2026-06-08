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

// nonEmptyChoiceCount 统计有效选项数（忽略空字符串槽位）。
func nonEmptyChoiceCount(choices []string) int {
	n := 0
	for _, c := range choices {
		if strings.TrimSpace(c) != "" {
			n++
		}
	}
	return n
}

// CoerceExerciseOutput 若 LLM 把选项写在题干里但未填 choices，自动转为 choice 题型。
func CoerceExerciseOutput(out *ExerciseOutput) {
	if out == nil {
		return
	}
	format := NormalizeAnswerFormat(out.AnswerFormat, out.QuestionType)
	if format == "choice" && nonEmptyChoiceCount(out.Choices) >= 2 {
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
		_, text, ok := choiceAtDisplayLetter(ex.Choices, L)
		if !ok {
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

// choiceAtDisplayLetter 将字母映射到选项文案：优先按 choices 下标（A→0），否则按跳过空项后的紧凑序号。
func choiceAtDisplayLetter(choices []string, letter rune) (idx int, text string, ok bool) {
	if slot := int(letter - 'A'); slot >= 0 && slot < len(choices) {
		if t := strings.TrimSpace(choices[slot]); t != "" {
			return slot, t, true
		}
	}
	n := 0
	for i, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if rune('A'+n) == letter {
			return i, c, true
		}
		n++
	}
	return 0, "", false
}

func formatChoicesForPrompt(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【选项对照表】（批改时字母必须与下表一致，禁止凭记忆编造）\n")
	n := 0
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		fmt.Fprintf(&b, "%c. %s\n", 'A'+n, c)
		n++
	}
	return strings.TrimRight(b.String(), "\n")
}

func normalizeCorrectAnswer(out ExerciseOutput, choiceMode string) ([]rune, bool) {
	mode := normalizeChoiceMode(choiceMode)
	if mode == "multiple" {
		if letters := normalizeLetterList(out.CorrectChoices); len(letters) > 0 {
			return letters, true
		}
		if letters := normalizeLetterList([]string{out.CorrectChoice}); len(letters) > 0 {
			return letters, true
		}
		return nil, false
	}
	if letters := normalizeLetterList([]string{out.CorrectChoice}); len(letters) > 0 {
		return letters, true
	}
	if letters := normalizeLetterList(out.CorrectChoices); len(letters) == 1 {
		return letters, true
	}
	return nil, false
}

func correctLettersFromContext(ex *storage.ExerciseContext) []rune {
	if ex == nil || ex.AnswerFormat != "choice" {
		return nil
	}
	if ex.ChoiceMode == "multiple" {
		if letters := normalizeLetterList(ex.CorrectChoices); len(letters) > 0 {
			return letters
		}
	}
	if letters := normalizeLetterList([]string{ex.CorrectChoice}); len(letters) > 0 {
		return letters
	}
	return normalizeLetterList(ex.CorrectChoices)
}

func normalizeLetterList(items []string) []rune {
	seen := make(map[rune]struct{})
	var letters []rune
	for _, item := range items {
		for _, L := range extractChoiceLetters(strings.TrimSpace(item)) {
			if _, dup := seen[L]; dup {
				continue
			}
			seen[L] = struct{}{}
			letters = append(letters, L)
		}
	}
	sortRunes(letters)
	return letters
}

func runesToLetterStrings(letters []rune) []string {
	out := make([]string, len(letters))
	for i, L := range letters {
		out[i] = string(L)
	}
	return out
}

func sortRunes(r []rune) {
	for i := 0; i < len(r); i++ {
		for j := i + 1; j < len(r); j++ {
			if r[j] < r[i] {
				r[i], r[j] = r[j], r[i]
			}
		}
	}
}

func letterSetsEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]rune(nil), a...)
	bb := append([]rune(nil), b...)
	sortRunes(aa)
	sortRunes(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

// parseUserChoiceLetters 从规范化后的用户作答中解析选项字母。
func parseUserChoiceLetters(userMsg string) []rune {
	if idx := strings.Index(userMsg, "我选择："); idx >= 0 {
		rest := userMsg[idx+len("我选择："):]
		parts := strings.Split(rest, "；")
		var letters []rune
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			runes := []rune(p)
			if runes[0] >= 'A' && runes[0] <= 'D' {
				letters = append(letters, runes[0])
			}
		}
		if len(letters) > 0 {
			sortRunes(letters)
			return letters
		}
	}
	letters := extractChoiceLetters(userMsg)
	sortRunes(letters)
	return letters
}

// GradeChoiceAnswer 选择题程序判分；无法判分时 Ok=false，由 LLM 兜底。
func GradeChoiceAnswer(ex *storage.ExerciseContext, userMsg string) (verdict ChoiceGradeVerdict, ok bool) {
	if ex == nil || ex.AnswerFormat != "choice" {
		return ChoiceGradeVerdict{}, false
	}
	correct := correctLettersFromContext(ex)
	if len(correct) == 0 {
		return ChoiceGradeVerdict{}, false
	}
	user := parseUserChoiceLetters(userMsg)
	if len(user) == 0 {
		return ChoiceGradeVerdict{}, false
	}
	passed := false
	if ex.ChoiceMode == "multiple" {
		passed = letterSetsEqual(user, correct)
	} else {
		passed = len(user) == 1 && len(correct) >= 1 && user[0] == correct[0]
	}
	return ChoiceGradeVerdict{
		Passed:         passed,
		UserLetters:    user,
		CorrectLetters: correct,
	}, true
}

func formatChoiceGradeVerdict(v *ChoiceGradeVerdict) string {
	if v == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("【系统判定】（选择题已由程序比对标准答案，passed 必须与此一致）\n")
	fmt.Fprintf(&b, "标准答案：%s\n", formatLetterList(v.CorrectLetters))
	fmt.Fprintf(&b, "用户选择：%s\n", formatLetterList(v.UserLetters))
	if v.Passed {
		b.WriteString("判定：正确\n")
	} else {
		b.WriteString("判定：错误\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatLetterList(letters []rune) string {
	if len(letters) == 0 {
		return "（无）"
	}
	parts := make([]string, len(letters))
	for i, L := range letters {
		parts[i] = string(L)
	}
	return strings.Join(parts, "、")
}
