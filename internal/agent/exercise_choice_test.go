package agent

import (
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

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
