package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const profileInitTimeout = 60 * time.Second

// InitProfileFromOnboarding 将引导问卷压成结构化全局画像并落库。
func (c *Coach) InitProfileFromOnboarding(ctx context.Context, userID, role, background, goal string) (string, error) {
	if c == nil || !c.llmClient(ctx).Configured() {
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
		TaskInstruction: "请根据【冷启动问卷】生成 background、goal、preference；不要写按课进展散文。",
		UserMessage:     b.String(),
		Phase:           "onboarding",
	}
	schema, _ := domain.LoadSchema("profile_init.json")
	msgs := c.prompter.BuildMessages(in, TaskProfileInit, schema)
	ctx = observability.WithGeneration(ctx, TaskProfileInit.GenerationName())

	var out ProfileGlobalOutput
	if err := c.llmClient(ctx).ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return "", err
	}
	bg := strings.TrimSpace(out.Background)
	if bg == "" {
		bg = role + "，" + background
	}
	g := strings.TrimSpace(out.Goal)
	if g == "" {
		g = goal
	}
	if err := c.store.WriteGlobalProfile(userID, bg, g, out.Preference); err != nil {
		return "", err
	}
	u, err := c.store.GetUser(userID)
	if err != nil {
		return "", err
	}
	return u.ProfileSummary, nil
}
