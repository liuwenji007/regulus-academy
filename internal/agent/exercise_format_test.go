package agent

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestNormalizeAnswerFormat(t *testing.T) {
	tests := []struct {
		format, qType, want string
	}{
		{"json", "", "json"},
		{"choice", "", "choice"},
		{"", "code_fill", "text"},
		{"", "bug_find", "text"},
		{"", "short_answer", "text"},
		{"", "unknown", "text"},
	}
	for _, tc := range tests {
		if got := NormalizeAnswerFormat(tc.format, tc.qType); got != tc.want {
			t.Fatalf("NormalizeAnswerFormat(%q,%q)=%q want %q", tc.format, tc.qType, got, tc.want)
		}
	}
}

func TestCoerceAnswerFormatForQuestion(t *testing.T) {
	tests := []struct {
		format, qType, question, want string
	}{
		{"json", "code_fill", "完成 TODO：declare module 并导出 createTool", "text"},
		{"json", "code_fill", "补全 docker-compose 中 depends_on 与 volumes", "json"},
		{"json", "code_fill", "补全以下 JSON 配置，填入 scripts 字段", "json"},
		{"json", "code_fill", "补全以下 JSON，请以 JSON 格式给出缺失字段", "json"},
		{"json", "short_answer", "说明 JSON 和 YAML 的区别", "text"},
		{"json", "short_answer", "请以 JSON 格式给出：1. 修改后的注释 2. 补全快捷键 3. 企业风险", "text"},
		{"json", "short_answer", "docker-compose 和 docker compose 有什么区别", "text"},
		{"json", "short_answer", "You are using Copilot. Answer in JSON format covering three points.", "text"},
	}
	for _, tc := range tests {
		got := CoerceAnswerFormatForQuestion(tc.format, tc.qType, tc.question)
		if got != tc.want {
			t.Fatalf("CoerceAnswerFormatForQuestion(%q,%q,%q)=%q want %q",
				tc.format, tc.qType, tc.question, got, tc.want)
		}
	}
}

func TestBuildExerciseContext_coercesCodeFillJSON(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "补全 TypeScript declare 全局变量",
		QuestionType: "code_fill",
		AnswerFormat: "json",
	})
	if ex.AnswerFormat != "text" {
		t.Fatalf("want text, got %s", ex.AnswerFormat)
	}
}

func TestExerciseMetaFromContext_includesQuestionType(t *testing.T) {
	ex := &storage.ExerciseContext{
		AnswerFormat: "text",
		QuestionType: "code_fill",
		Question:     "补全 filter",
	}
	meta := exerciseMetaFromContext(ex)
	if meta == nil || meta.QuestionType != "code_fill" {
		t.Fatalf("expected questionType code_fill, got %+v", meta)
	}
	if meta.AnswerFormat != "text" {
		t.Fatalf("answerFormat=%q", meta.AnswerFormat)
	}
}

func TestBuildExerciseContextChoiceFallback(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"仅一个"},
	})
	if ex.AnswerFormat != "text" {
		t.Fatalf("expected text fallback, got %s", ex.AnswerFormat)
	}
	if ex.ChoiceMode != "" {
		t.Fatalf("text fallback should clear choiceMode, got %q", ex.ChoiceMode)
	}
	if len(ex.Choices) != 0 {
		t.Fatal("text fallback should clear choices")
	}
}

func TestBuildExerciseContextChoiceFallback_sparseEmptySlot(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"option1", ""},
	})
	if ex.AnswerFormat != "text" {
		t.Fatalf("expected text fallback for sparse single option, got %s", ex.AnswerFormat)
	}
	if len(ex.Choices) != 0 {
		t.Fatal("text fallback should clear choices")
	}
}

func TestBuildExerciseContextChoice_keepsSparseWithTwoNonEmpty(t *testing.T) {
	ex := BuildExerciseContext(ExerciseOutput{
		Question:     "选一项",
		AnswerFormat: "choice",
		Choices:      []string{"甲", "", "乙"},
	})
	if ex.AnswerFormat != "choice" || len(ex.Choices) != 3 {
		t.Fatalf("want choice with sparse slots, got format=%s choices=%v", ex.AnswerFormat, ex.Choices)
	}
}
