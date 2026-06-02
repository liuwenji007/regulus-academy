package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
)

type navIntentLLMOutput struct {
	Action    string `json:"action"`
	CourseRef string `json:"course_ref"`
	NodeRef   string `json:"node_ref"`
	ReplyHint string `json:"reply_hint"`
}

// ParseNavIntent 用 LLM 解析模糊导航意图（规则未命中时兜底）
func ParseNavIntent(ctx context.Context, client llm.Provider, ctxNav navContext, userText string) (NavigationIntent, error) {
	if client == nil || !client.Configured() {
		return NavigationIntent{}, fmt.Errorf("未配置 LLM")
	}
	schema, err := domain.LoadSchema("channel_nav.json")
	if err != nil {
		return NavigationIntent{}, err
	}
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy IM 导航意图分析器。根据用户消息和上下文，判断其想查看课程、进入某课、开始某节点、续学或看进度。只输出 JSON，不要解释。"},
		{Role: "user", Content: buildNavIntentPrompt(ctxNav, userText) + "\n\n输出 JSON Schema：\n" + schema},
	}
	var out navIntentLLMOutput
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return NavigationIntent{}, fmt.Errorf("导航意图分析失败: %w", err)
	}
	return normalizeNavIntentLLM(out), nil
}

func buildNavIntentPrompt(ctx navContext, userText string) string {
	var b strings.Builder
	b.WriteString("【用户消息】\n")
	b.WriteString(userText)
	b.WriteString("\n\n【上下文】\n")
	if ctx.HasActiveSession {
		b.WriteString("- 有进行中的学习会话\n")
	}
	if ctx.ActiveDomainID != "" {
		b.WriteString(fmt.Sprintf("- 当前活跃课程 domain_id=%s", ctx.ActiveDomainID))
		if ctx.ActiveNodeKey != "" {
			b.WriteString(fmt.Sprintf(" 节点=%s", ctx.ActiveNodeKey))
		}
		b.WriteString("\n")
	}
	if ctx.PendingDomainID != "" {
		b.WriteString(fmt.Sprintf("- 已选课程待选节点 domain_id=%s\n", ctx.PendingDomainID))
	}
	if len(ctx.Courses) == 0 {
		b.WriteString("- 尚无课程（用户需在 Web 端建课）\n")
	} else {
		b.WriteString("【课程列表】\n")
		for i, d := range ctx.Courses {
			slug := d.Slug
			if slug == "" {
				slug = "-"
			}
			b.WriteString(fmt.Sprintf("%d. %s slug=%s 进度=%d/%d\n", i+1, d.Name, slug, d.Completed, d.NodeTotal))
		}
	}
	if len(ctx.FlatNodes) > 0 {
		b.WriteString("【当前课程节点】\n")
		for i, n := range ctx.FlatNodes {
			b.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, n.Title, n.Key))
		}
	}
	b.WriteString("\n若无法确定课程或节点，action 用 clarify 并在 reply_hint 中简短追问。")
	return b.String()
}

func normalizeNavIntentLLM(out navIntentLLMOutput) NavigationIntent {
	action := NavAction(strings.TrimSpace(out.Action))
	switch action {
	case NavListCourses, NavShowNodes, NavStartNode, NavContinue, NavProgress, NavHelp, NavClarify:
	default:
		action = NavClarify
	}
	return NavigationIntent{
		Action:    action,
		CourseRef: strings.TrimSpace(out.CourseRef),
		NodeRef:   strings.TrimSpace(out.NodeRef),
		ReplyHint: strings.TrimSpace(out.ReplyHint),
	}
}
