package agent

import (
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestFormatChoiceLabel_skipsDuplicatePrefix(t *testing.T) {
	got := formatChoiceLabel('C', "C. if 语句支持初始化")
	if got != "C. if 语句支持初始化" {
		t.Fatalf("got %q", got)
	}
	got = formatChoiceLabel('A', "统一战线")
	if got != "A. 统一战线" {
		t.Fatalf("got %q", got)
	}
	// 文案以别的字母开头时仍应拼接当前字母
	got = formatChoiceLabel('B', "A. 看起来像前缀但不是本项")
	if got != "B. A. 看起来像前缀但不是本项" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandChoiceAnswer_skipsDuplicatePrefix(t *testing.T) {
	ex := &storage.ExerciseContext{
		AnswerFormat: "choice",
		ChoiceMode:   "single",
		Choices: []string{
			"A. if 不能带初始化语句",
			"B. switch 不支持 fallthrough",
			"C. if 语句支持在条件部分使用初始化语句",
		},
	}
	got := ExpandChoiceAnswer(ex, "C")
	if got != "我选择：C. if 语句支持在条件部分使用初始化语句" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "C. C.") {
		t.Fatalf("double prefix: %q", got)
	}
}

func TestParseLetteredChoices(t *testing.T) {
	q := `毛泽东在《〈共产党人〉发刊词》中总结了三大法宝。以下哪一项**不属于**三大法宝？

A. 统一战线
B. 武装斗争
C. 独立自主
D. 党的建设`
	stem, choices, ok := ParseLetteredChoices(q)
	if !ok || len(choices) != 4 {
		t.Fatalf("parse failed: ok=%v choices=%v", ok, choices)
	}
	if choices[1] != "武装斗争" {
		t.Fatalf("B=%q", choices[1])
	}
	if strings.Contains(stem, "A. 统一战线") {
		t.Fatal("stem should not contain option lines")
	}
}

func TestNonEmptyChoiceCount(t *testing.T) {
	if nonEmptyChoiceCount([]string{"a", "", "b"}) != 2 {
		t.Fatal("should count non-empty only")
	}
	if nonEmptyChoiceCount([]string{"only", ""}) != 1 {
		t.Fatal("sparse single option")
	}
}

func TestCoerceExerciseOutput_sparseSingleChoiceDoesNotStick(t *testing.T) {
	out := &ExerciseOutput{
		Question:     "以下哪项正确？",
		AnswerFormat: "choice",
		Choices:      []string{"option1", ""},
		QuestionType: "short_answer",
	}
	CoerceExerciseOutput(out)
	ex := BuildExerciseContext(*out)
	if ex.AnswerFormat != "text" {
		t.Fatalf("want text fallback, got %s", ex.AnswerFormat)
	}
}

func TestCoerceExerciseOutput(t *testing.T) {
	out := &ExerciseOutput{
		Question: `以下哪一项不属于三大法宝？
A. 统一战线
B. 武装斗争
C. 独立自主
D. 党的建设`,
		QuestionType: "short_answer",
		AnswerFormat: "text",
	}
	CoerceExerciseOutput(out)
	if out.AnswerFormat != "choice" || len(out.Choices) != 4 {
		t.Fatalf("coerce: format=%s choices=%d", out.AnswerFormat, len(out.Choices))
	}
}

func TestExpandChoiceAnswer(t *testing.T) {
	ex := &storage.ExerciseContext{
		AnswerFormat: "choice",
		ChoiceMode:   "single",
		Choices:      []string{"统一战线", "武装斗争", "独立自主", "党的建设"},
	}
	got := ExpandChoiceAnswer(ex, "B")
	if !strings.Contains(got, "武装斗争") || !strings.Contains(got, "B.") {
		t.Fatalf("expand: %q", got)
	}
}

func TestParseLetteredChoices_outOfDocumentOrder(t *testing.T) {
	q := `Pick one:
D. fourth
B. second
A. first`
	_, choices, ok := ParseLetteredChoices(q)
	if !ok || len(choices) != 4 {
		t.Fatalf("parse: ok=%v len=%d choices=%v", ok, len(choices), choices)
	}
	if choices[0] != "first" || choices[1] != "second" || choices[3] != "fourth" {
		t.Fatalf("choices by letter index: %v", choices)
	}
}

func TestFormatChoicesForPrompt_skipsEmptyWithCompactLetters(t *testing.T) {
	got := formatChoicesForPrompt([]string{"First", "", "Third"})
	if !strings.Contains(got, "A. First") || !strings.Contains(got, "B. Third") {
		t.Fatalf("compact letters: %q", got)
	}
	if strings.Contains(got, "C. Third") {
		t.Fatalf("should not use sparse index letter: %q", got)
	}
}

func TestChoiceAtDisplayLetter_sparseAndCompact(t *testing.T) {
	choices := []string{"first", "second", "", "fourth"}
	_, text, ok := choiceAtDisplayLetter(choices, 'D')
	if !ok || text != "fourth" {
		t.Fatalf("sparse D: ok=%v text=%q", ok, text)
	}
	_, text, ok = choiceAtDisplayLetter(choices, 'C')
	if !ok || text != "fourth" {
		t.Fatalf("compact C: ok=%v text=%q", ok, text)
	}
}

func TestExpandChoiceAnswer_outOfDocumentOrderChoices(t *testing.T) {
	out := &ExerciseOutput{
		Question: `Pick one:
D. fourth
B. second
A. first`,
		QuestionType: "short_answer",
		AnswerFormat: "text",
	}
	CoerceExerciseOutput(out)
	ex := BuildExerciseContext(*out)
	got := ExpandChoiceAnswer(ex, "A")
	if !strings.Contains(got, "A. first") || strings.Contains(got, "fourth") {
		t.Fatalf("expand A should map to first: %q", got)
	}
}

func TestBuildExerciseContext_storesCorrectChoice(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:      "以下哪项正确？",
		QuestionType:  "short_answer",
		AnswerFormat:  "choice",
		Choices:       []string{"只有 1、2 正确", "只有 1、2、4 正确", "全部正确", "都不正确"},
		ChoiceMode:    "single",
		CorrectChoice: "B",
	})
	if ex.CorrectChoice != "B" {
		t.Fatalf("correctChoice=%q", ex.CorrectChoice)
	}
}

func TestGradeChoiceAnswer_compositeSingleCorrect(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:      "关于 Hook 与路由，以下说法哪些正确？\n1. beforeRoute...\n2. Multi-Agent...\n3. afterForward...\n4. onError...",
		QuestionType:  "short_answer",
		AnswerFormat:  "choice",
		Choices:       []string{"只有 1、2 正确", "只有 1、2、4 正确", "只有 2、3 正确", "全部正确"},
		ChoiceMode:    "single",
		CorrectChoice: "B",
	})
	user := ExpandChoiceAnswer(ex, "B")
	v, ok := GradeChoiceAnswer(ex, user)
	if !ok {
		t.Fatal("expected programmatic grade")
	}
	if !v.Passed {
		t.Fatalf("expected pass, got verdict=%+v", v)
	}
}

func TestGradeChoiceAnswer_compositeSingleWrong(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:      "以下说法哪些正确？",
		QuestionType:  "short_answer",
		AnswerFormat:  "choice",
		Choices:       []string{"只有 1、2 正确", "只有 1、2、4 正确"},
		ChoiceMode:    "single",
		CorrectChoice: "B",
	})
	user := ExpandChoiceAnswer(ex, "A")
	v, ok := GradeChoiceAnswer(ex, user)
	if !ok || v.Passed {
		t.Fatalf("expected fail, ok=%v verdict=%+v", ok, v)
	}
}

func TestGradeChoiceAnswer_multiple(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:       "多选",
		QuestionType:   "short_answer",
		AnswerFormat:   "choice",
		Choices:        []string{"a", "b", "c", "d"},
		ChoiceMode:     "multiple",
		CorrectChoices: []string{"A", "C"},
	})
	user := "我选择：A. a；C. c"
	v, ok := GradeChoiceAnswer(ex, user)
	if !ok || !v.Passed {
		t.Fatalf("expected pass, ok=%v verdict=%+v", ok, v)
	}
}

func TestGradeChoiceAnswer_noCorrectAnswerFallback(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "以下哪项？",
		QuestionType: "short_answer",
		AnswerFormat: "choice",
		Choices:      []string{"x", "y"},
		ChoiceMode:   "single",
	})
	_, ok := GradeChoiceAnswer(ex, "我选择：A. x")
	if ok {
		t.Fatal("expected fallback when no correct answer stored")
	}
}

func TestFormatChoiceGradeVerdict(t *testing.T) {
	got := formatChoiceGradeVerdict(&ChoiceGradeVerdict{
		Passed:         true,
		UserLetters:    []rune{'B'},
		CorrectLetters: []rune{'B'},
	}, true)
	if !strings.Contains(got, "判定：正确") || !strings.Contains(got, "标准答案：B") {
		t.Fatalf("verdict prompt: %q", got)
	}
	firstMiss := formatChoiceGradeVerdict(&ChoiceGradeVerdict{
		Passed:         false,
		UserLetters:    []rune{'A'},
		CorrectLetters: []rune{'B'},
	}, false)
	if strings.Contains(firstMiss, "标准答案：") || !strings.Contains(firstMiss, "禁止") {
		t.Fatalf("first miss should hide answer: %q", firstMiss)
	}
	secondMiss := formatChoiceGradeVerdict(&ChoiceGradeVerdict{
		Passed:         false,
		UserLetters:    []rune{'A'},
		CorrectLetters: []rune{'B'},
	}, true)
	if !strings.Contains(secondMiss, "标准答案：B") {
		t.Fatalf("second miss may reveal: %q", secondMiss)
	}
}
