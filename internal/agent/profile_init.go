package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const profileInitTimeout = 60 * time.Second

// InitProfileFromOnboarding 将引导问卷压成 profile_summary 并落库。
func (c *Coach) InitProfileFromOnboarding(ctx context.Context, userID, role, background, goal string) (string, error) {
	if c == nil || !c.llmClient().Configured() {
		return "", fmt.Errorf("未配置 LLM，无法生成学生画像")
	}
	role = strings.TrimSpace(role)
	background = strings.TrimSpace(background)
	goal = strings.TrimSpace(goal)
	if role == "" || background == "" {
		return "", fmt.Errorf("身份与已有基础不能为空")
	}

	ctx, cancel := context.WithTimeout(ctx, profileInitTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name:   "coach.profile_init",
		UserID: userID,
	})
	defer endTrace()

	var b strings.Builder
	b.WriteString("【冷启动问卷】\n")
	b.WriteString("身份/角色：")
	b.WriteString(role)
	b.WriteString("\n已有基础：")
	b.WriteString(background)
	if goal != "" {
		b.WriteString("\n学习目标：")
		b.WriteString(goal)
	}

	in := PromptInput{
		TaskInstruction: "请根据【冷启动问卷】生成首版学生画像 summary。",
		UserMessage:     b.String(),
		Phase:           "onboarding",
	}
	schema, _ := domain.LoadSchema("profile_init.json")
	msgs := c.prompter.BuildMessages(in, TaskProfileInit, schema)
	ctx = observability.WithGeneration(ctx, TaskProfileInit.GenerationName())

	var out ProfileRefreshOutput
	if err := c.llmClient().ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return "", err
	}
	summary := strings.TrimSpace(out.Summary)
	if summary == "" {
		return "", fmt.Errorf("模型未返回有效画像")
	}
	if utf8.RuneCountInString(summary) > maxProfileSummaryRunes {
		summary = truncateRunes(summary, maxProfileSummaryRunes)
	}
	if err := c.store.UpdateUserProfileSummary(userID, summary); err != nil {
		return "", err
	}
	return summary, nil
}
