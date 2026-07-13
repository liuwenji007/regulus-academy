package agent

import "testing"

func TestSanitizeExerciseQuestion_stripsBacktickExample(t *testing.T) {
	in := "请将四个空依次填写（用空格分隔即可，如 `Partial Pick Exclude 'error'|'warning'`）："
	want := "请将四个空依次填写（用空格分隔即可）："
	if got := SanitizeExerciseQuestion(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSanitizeExerciseQuestion_stripsStandaloneExample(t *testing.T) {
	in := "补全类型定义。\n\n如 `Partial<Pick<User, 'name'>>` 的写法。"
	want := "补全类型定义。"
	if got := SanitizeExerciseQuestion(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSanitizeExerciseQuestion_keepsNormalQuestion(t *testing.T) {
	in := "UpdateUserPayload 应只允许修改 name 和 email，且全部可选。请写出类型定义。"
	if got := SanitizeExerciseQuestion(in); got != in {
		t.Fatalf("should not change: %q", got)
	}
}

func TestBuildExerciseContext_sanitizesQuestion(t *testing.T) {
	out := ExerciseOutput{
		Question:     "填空（如 `foo bar`）",
		QuestionType: "code_fill",
		AnswerFormat: "text",
	}
	ex := BuildExerciseContext(out)
	if ex == nil || ex.Question != "填空" {
		t.Fatalf("question=%q", ex.Question)
	}
}
