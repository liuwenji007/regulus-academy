package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const profileMergeTimeout = 60 * time.Second

// RefineUserProfile 将用户补充合并进全局画像并落库。
func (c *Coach) RefineUserProfile(ctx context.Context, userID, supplement string) (string, error) {
	if c == nil || !c.llmClient(ctx).Configured() {
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

	ctx, cancel := context.WithTimeout(ctx, profileMergeTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name:   "coach.profile_merge",
		UserID: userID,
	})
	defer endTrace()

	var b strings.Builder
	global := storage.FormatStructuredGlobal(user.ProfileBackground, user.ProfileGoal, user.ProfilePreference)
	if global == "" {
		global = storage.ParseBackgroundGoal(user.ProfileSummary)
	}
	if global != "" {
		b.WriteString("【当前学生画像】\n")
		b.WriteString(global)
		b.WriteString("\n\n")
	}
	b.WriteString("【用户补充】\n")
	b.WriteString(supplement)

	in := PromptInput{
		TaskInstruction: "请根据【当前学生画像】与【用户补充】输出合并后的 background、goal、preference；禁止写入按课课堂细节。",
		UserMessage:     b.String(),
		Phase:           "settings",
		UserProfile:     global,
	}
	schema, _ := domain.LoadSchema("profile_merge.json")
	msgs := c.prompter.BuildMessages(in, TaskProfileMerge, schema)
	ctx = observability.WithGeneration(ctx, TaskProfileMerge.GenerationName())

	var out ProfileGlobalOutput
	if err := c.llmClient(ctx).ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Background) == "" && strings.TrimSpace(out.Goal) == "" {
		return "", fmt.Errorf("模型未返回有效画像")
	}
	if err := c.store.WriteGlobalProfile(userID, out.Background, out.Goal, out.Preference); err != nil {
		return "", err
	}
	u, err := c.store.GetUser(userID)
	if err != nil {
		return "", err
	}
	return u.ProfileSummary, nil
}
