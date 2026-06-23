package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const navConfidenceMin = 0.75

type navIntentLLMOutput struct {
	Action     string  `json:"action"`
	CourseRef  string  `json:"course_ref"`
	NodeRef    string  `json:"node_ref"`
	ReplyHint  string  `json:"reply_hint"`
	Confidence float64 `json:"confidence"`
}

// ParseNavIntent 用 LLM 解析模糊导航意图（规则未命中时兜底）
func ParseNavIntent(ctx context.Context, client llm.Provider, ctxNav navContext, userText string) (NavigationIntent, error) {
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "channel.nav", UserID: ctxNav.UserID, Channel: ctxNav.Platform, Input: userText,
	})
	defer endTrace()

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
	ctx = observability.WithGeneration(ctx, "channel.nav")
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
	b.WriteString(`
规则：
- confidence：0.0～1.0，表示对导航意图的把握程度
- 若无法确定课程、节点或动作，confidence 应低于 0.75，action 用 clarify，并在 reply_hint 中简短追问
- 把握充分时 confidence 应 ≥ 0.75`)
	return b.String()
}

func normalizeNavIntentLLM(out navIntentLLMOutput) NavigationIntent {
	action := NavAction(strings.TrimSpace(out.Action))
	switch action {
	case NavListCourses, NavShowNodes, NavStartNode, NavContinue, NavProgress, NavHelp, NavClarify:
	default:
		action = NavClarify
	}
	replyHint := strings.TrimSpace(out.ReplyHint)
	if out.Confidence < navConfidenceMin {
		original := action
		action = NavClarify
		if replyHint == "" {
			replyHint = defaultNavClarifyMessage(original, out)
		}
	}
	return NavigationIntent{
		Action:    action,
		CourseRef: strings.TrimSpace(out.CourseRef),
		NodeRef:   strings.TrimSpace(out.NodeRef),
		ReplyHint: replyHint,
	}
}

func defaultNavClarifyMessage(original NavAction, out navIntentLLMOutput) string {
	switch original {
	case NavStartNode, NavShowNodes:
		if strings.TrimSpace(out.CourseRef) != "" || strings.TrimSpace(out.NodeRef) != "" {
			return "我不太确定你想进入哪门课或哪个节点，请再说具体一些，或发送「我的课程」查看列表。"
		}
	case NavContinue:
		return "你是想继续上次的学习，还是查看课程列表？"
	case NavProgress:
		return "你想查看哪门课的进度？也可以说「我的课程」先看列表。"
	}
	return "请说明你想查看哪门课程或哪个节点，也可以说「我的课程」查看列表。"
}
