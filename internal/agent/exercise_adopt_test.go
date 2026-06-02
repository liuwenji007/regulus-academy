package agent

import "testing"

func TestParseExerciseJSONText(t *testing.T) {
	raw := `好的，进入练习环节：
{"question":"1+1=?","question_type":"short_answer","answer_format":"text","reinforced_concepts":[]}`
	out, ok := parseExerciseJSONText(raw)
	if !ok || out.Question != "1+1=?" || out.AnswerFormat != "text" {
		t.Fatalf("parse failed: ok=%v out=%+v", ok, out)
	}
}
