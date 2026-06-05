package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const profileMergeTimeout = 60 * time.Second

// RefineUserProfile 将用户补充合并进现有画像并落库。
func (c *Coach) RefineUserProfile(ctx context.Context, userID, supplement string) (string, error) {
	if c == nil || !c.llmClient().Configured() {
		return "", fmt.Errorf("未配置 LLM，无法合并学生画像")
	}
	supplement = strings.TrimSpace(supplement)
	if supplement == "" {
		return "", fmt.Errorf("补充内容不能为空")
	}
	user, err := c.store.GetUser(userID)
	if err != nil {
		return "", err
	}
	existing := strings.TrimSpace(user.ProfileSummary)

	ctx, cancel := context.WithTimeout(ctx, profileMergeTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name:   "coach.profile_merge",
		UserID: userID,
	})
	defer endTrace()

	var b strings.Builder
	if existing != "" {
		b.WriteString("【当前学生画像】\n")
		b.WriteString(existing)
		b.WriteString("\n\n")
	}
	b.WriteString("【用户补充】\n")
	b.WriteString(supplement)

	in := PromptInput{
		TaskInstruction: "请根据【当前学生画像】与【用户补充】输出合并后的 summary。",
		UserMessage:     b.String(),
		Phase:           "settings",
		UserProfile:     existing,
	}
	schema, _ := domain.LoadSchema("profile_merge.json")
	msgs := c.prompter.BuildMessages(in, TaskProfileMerge, schema)
	ctx = observability.WithGeneration(ctx, TaskProfileMerge.GenerationName())

	var out ProfileRefreshOutput
	if err := c.llmClient().ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return "", err
	}
	summary := strings.TrimSpace(out.Summary)
	if summary == "" {
		return "", fmt.Errorf("模型未返回有效画像")
	}
	if err := WriteUserProfile(c.store, userID, summary); err != nil {
		return "", err
	}
	return summary, nil
}
