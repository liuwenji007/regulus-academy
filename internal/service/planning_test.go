package service

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/llm"
)

func TestPlanningShouldSynth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		phase   string
		msg     string
		wantSyn bool
	}{
		{"intake", "最近项目很多，有点焦虑", false},
		{"intake", "帮我整理出行动方案", true},
		{"plan_ready", "我还得准备个汇报", false},
		{"plan_ready", "帮我把今日行动减到 2 条", true},
		{"plan_ready", "更新规划，优先写周报", true},
	}
	for _, c := range cases {
		got := agent.WantsExplicitPlan(c.msg) ||
			(c.phase == "plan_ready" && agent.WantsPlanUpdate(c.msg))
		if got != c.wantSyn {
			t.Fatalf("phase=%s msg=%q want=%v got=%v", c.phase, c.msg, c.wantSyn, got)
		}
	}
}

func TestIntakeReadySynthesizeHistoryIncludesIntakeReply(t *testing.T) {
	t.Parallel()
	prior := []llm.Message{{Role: "assistant", Content: "你好，今天想梳理什么？"}}
	hist := intakeReadySynthesizeHistory(prior, "帮我出行动方案", "好的，我先帮你归纳现状。")
	if len(hist) != 3 {
		t.Fatalf("len=%d", len(hist))
	}
	if hist[1].Role != "user" || hist[1].Content != "帮我出行动方案" {
		t.Fatalf("user msg: %+v", hist[1])
	}
	if hist[2].Role != "assistant" || hist[2].Content != "好的，我先帮你归纳现状。" {
		t.Fatalf("intake reply: %+v", hist[2])
	}
}
