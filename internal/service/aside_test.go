package service

import (
	"strings"
	"testing"
)

func TestExtractPrerequisiteConcepts(t *testing.T) {
	answer := `### 解释

一段正文。

**可能需要先了解** 事务、幂等、哈希表

— 回到主线`
	got := extractPrerequisiteConcepts(answer)
	want := map[string]bool{"事务": true, "幂等": true, "哈希表": true}
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected %q", c)
		}
	}
}

func TestExtractPrerequisiteConcepts_NoFallbackToQuestion(t *testing.T) {
	// 无「可能需要」行时不抽概念（Ask 也不再把整句问题落库）
	got := extractPrerequisiteConcepts("RPC 是远程过程调用。回到主线继续学。")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestExtractPrerequisiteConcepts_FiltersNoise(t *testing.T) {
	answer := "可能需要先了解 概念、a、幂等性"
	got := extractPrerequisiteConcepts(answer)
	joined := strings.Join(got, ",")
	if strings.Contains(joined, "概念") || strings.Contains(joined, "a") {
		t.Fatalf("noise leaked: %v", got)
	}
	found := false
	for _, c := range got {
		if c == "幂等" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 幂等, got %v", got)
	}
}
