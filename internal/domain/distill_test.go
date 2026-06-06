package domain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDistillMapReduce(t *testing.T) {
	mock := &seqMockLLM{
		replies: []string{
			mustJSON(distillMapOutput{Points: []string{"要点 A"}, Concepts: []string{"概念 A"}}),
			mustJSON(DistillOutline{
				Title: "测试主题",
				Sections: []DistillSection{{
					Heading: "第一章", Points: []string{"要点 A"}, Concepts: []string{"概念 A"},
				}},
				ScopeBreadth: ScopeModerate,
			}),
		},
	}
	outline, err := Distill(context.Background(), mock, "这是一段测试材料，讲述 goroutine 与 channel。")
	if err != nil {
		t.Fatal(err)
	}
	if outline.Title != "测试主题" {
		t.Fatalf("title=%q", outline.Title)
	}
	formatted := FormatRefOutline(outline)
	if formatted == "" ||
		!strings.Contains(formatted, "参考材料大纲") ||
		!strings.Contains(formatted, "测试主题") ||
		!strings.Contains(formatted, "第一章") {
		t.Fatalf("formatted=%q", formatted)
	}
}

func TestFormatRefOutlineEmpty(t *testing.T) {
	if FormatRefOutline(nil) != "" {
		t.Fatal("nil outline 应返回空")
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
