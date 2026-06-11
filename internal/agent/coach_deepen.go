package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// deepenConcept 对单个概念做三拍深讲，并记入 ExplainedConcepts。
func (c *Coach) deepenConcept(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, concept string) (string, error) {
	concept = strings.TrimSpace(concept)
	if concept == "" {
		return "", fmt.Errorf("深讲目标为空")
	}
	in, err := c.buildInput(sess,
		fmt.Sprintf("请简要补讲概念：%s（2～4 句）", concept),
		"")
	if err != nil {
		return "", err
	}
	in.DeepenTarget = concept
	in.ExplainedConcepts = sctx.ExplainedConcepts
	msgs := c.prompter.BuildMessages(in, TaskDeepen, "")
	ctx = observability.WithGeneration(ctx, TaskDeepen.GenerationName())
	content, err := c.llmClient(ctx).ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return "", err
	}
	content = sanitizeCoachPlainText(content)
	node, _ := c.registry.GetNode(c.store, sess.DomainID, sess.DomainSlug, sess.NodeKey)
	var core []string
	if node != nil {
		core = node.CoreConcepts
	}
	MergeExplainedConcepts(sctx, core, []string{concept})
	return content, nil
}

// beginTaskInstruction 开场任务说明（正向、简短，避免堆禁令）。
func beginTaskInstruction(node *domain.NodeSpec) string {
	if node == nil || len(node.CoreConcepts) == 0 {
		return "请做当前节点的开场讲解，并邀请用户提问或回复「开始练习」。"
	}
	n := len(node.CoreConcepts)
	base := "请做当前节点的开场讲解，覆盖全部核心概念，适度展开让读者能听懂再练。"
	if n >= 2 {
		base += "概念较多时用 Markdown 分条（- **概念名**：…），每条 2～3 句。"
	}
	if n >= 5 {
		base += fmt.Sprintf("可先 1 句总览再分 %d 条。", n)
	}
	base += "最后邀请用户提问，或回复「开始练习」。"
	return base
}

// recordBeginExplained 标记开场已完成；不在此虚标 ExplainedConcepts（由答疑/补讲/练习后写入）。
func recordBeginExplained(sctx *storage.SessionContext, node *domain.NodeSpec) {
	if sctx == nil || node == nil {
		return
	}
	sctx.OverviewDone = true
}
