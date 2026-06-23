package channel

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeNavIntentLowConfidenceForcesClarify(t *testing.T) {
	intent := normalizeNavIntentLLM(navIntentLLMOutput{
		Action:     "list_courses",
		Confidence: 0.5,
	})
	if intent.Action != NavClarify {
		t.Fatalf("action=%q want clarify", intent.Action)
	}
	if intent.ReplyHint == "" {
		t.Fatal("expected default clarify message")
	}
}

func TestNormalizeNavIntentLowConfidenceKeepsReplyHint(t *testing.T) {
	intent := normalizeNavIntentLLM(navIntentLLMOutput{
		Action:     "start_node",
		CourseRef:  "Go",
		Confidence: 0.4,
		ReplyHint:  "你想学 Go 并发还是 Go 语言？",
	})
	if intent.Action != NavClarify {
		t.Fatalf("action=%q want clarify", intent.Action)
	}
	if intent.ReplyHint != "你想学 Go 并发还是 Go 语言？" {
		t.Fatalf("reply=%q", intent.ReplyHint)
	}
}

func TestNormalizeNavIntentHighConfidencePreservesAction(t *testing.T) {
	intent := normalizeNavIntentLLM(navIntentLLMOutput{
		Action:     "list_courses",
		Confidence: 0.85,
	})
	if intent.Action != NavListCourses {
		t.Fatalf("action=%q want list_courses", intent.Action)
	}
}

func TestNormalizeNavIntentLowConfidenceStartNodeDefaultHint(t *testing.T) {
	intent := normalizeNavIntentLLM(navIntentLLMOutput{
		Action:     "start_node",
		CourseRef:  "go",
		Confidence: 0.3,
	})
	if intent.Action != NavClarify {
		t.Fatalf("action=%q want clarify", intent.Action)
	}
	if !strings.Contains(intent.ReplyHint, "哪门课") {
		t.Fatalf("reply=%q", intent.ReplyHint)
	}
}

func TestRouterLLMNavIntentLowConfidence(t *testing.T) {
	mock := &navLLMMock{response: `{"action":"list_courses","course_ref":"","node_ref":"","reply_hint":"","confidence":0.5}`}
	router, _, _ := setupNavRouter(t, mock)

	result := router.Handle(context.Background(), MessageEvent{
		Platform: PlatformTelegram, PlatformUserID: "u-nav", Text: "那个课",
	})
	if len(result.Replies) == 0 {
		t.Fatal("expected clarify reply")
	}
	if strings.Contains(result.Replies[0], "你的课程：") {
		t.Fatalf("低置信度不应执行 list_courses: %q", result.Replies[0])
	}
	if !strings.Contains(result.Replies[0], "课程") {
		t.Fatalf("应追问用户: %q", result.Replies[0])
	}
}
