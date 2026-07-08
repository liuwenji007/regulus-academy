package service

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
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
