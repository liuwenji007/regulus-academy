package agent

import "testing"

func TestIsJSONSchemaDocument_exerciseSchema(t *testing.T) {
	t.Parallel()
	schema := `{
  "type": "object",
  "required": ["question", "question_type", "answer_format", "reinforced_concepts"],
  "properties": {
    "question": {"type": "string", "description": "题目正文"},
    "question_type": {"type": "string", "enum": ["code_fill", "bug_find", "short_answer"]},
    "answer_format": {"type": "string", "enum": ["text", "json", "choice"]},
    "reinforced_concepts": {"type": "array", "items": {"type": "string"}}
  }
}`
	if !IsJSONSchemaDocument(schema) {
		t.Fatal("exercise schema should be detected as schema echo")
	}
}

func TestIsJSONSchemaDocument_realExercise(t *testing.T) {
	t.Parallel()
	inst := `{"question":"说明 goroutine 与线程的区别","question_type":"short_answer","answer_format":"text","reinforced_concepts":["轻量级"]}`
	if IsJSONSchemaDocument(inst) {
		t.Fatal("real exercise instance must not be treated as schema")
	}
}

func TestValidateExerciseOutput(t *testing.T) {
	t.Parallel()
	if ValidateExerciseOutput(ExerciseOutput{}) == nil {
		t.Fatal("empty question should fail")
	}
	if err := ValidateExerciseOutput(ExerciseOutput{Question: "补全代码"}); err != nil {
		t.Fatal(err)
	}
}

func TestParseExerciseOutputRaw_rejectsSchema(t *testing.T) {
	t.Parallel()
	schema := `{"type":"object","required":["question"],"properties":{"question":{"type":"string"}}}`
	if _, ok := parseExerciseOutputRaw(schema); ok {
		t.Fatal("schema must not parse as exercise")
	}
}
