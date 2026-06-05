package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func wantsStartNext(msg string) bool {
	return domain.MatchTrigger("start_next", msg)
}

// blockStartNextUntilCompleted 节点未完成时拦截「下一节」，避免在对话里直接切节。
func (c *Coach) blockStartNextUntilCompleted(sess *storage.Session) *MessageResult {
	phase := sess.Phase
	if phase == "" {
		phase = "explain"
	}
	msg := "本节点尚未完成。请先完成当前练习或巩固薄弱点。"
	switch phase {
	case "exercise":
		msg += "你可以提交答案，或者说「不懂，回讲解」。"
	case "review":
		msg += "可以说「不懂，回讲解」，或点击「开始练习」再练一题。"
	default:
		msg += "可以说「开始练习」完成小测。"
	}
	msg += "\n\n若你认为本节点已够用，请说明「已经掌握，下一节」，我会评估后为你点亮本节点；点亮后在页面底部点击「继续 · 下一节」进入新讲解。"
	return &MessageResult{
		Role:    "assistant",
		Content: msg,
		Phase:   phase,
	}
}

func (c *Coach) startNextNode(ctx context.Context, completed *storage.Session) (*MessageResult, error) {
	tree, err := c.store.GetDomainTree(completed.UserID, completed.DomainID)
	if err != nil || tree == nil {
		return nil, fmt.Errorf("无法加载知识树")
	}
	nextKey, layer, title, ok := domain.NextNodeAfter(tree, completed.NodeKey)
	if !ok {
		return &MessageResult{
			Role:    "assistant",
			Content: "恭喜，本课程所有节点都已完成！可以在 Web 端查看整体进度。",
			Phase:   "completed",
		}, nil
	}

	slug, _ := c.store.GetDomainSlug(completed.DomainID)
	sctx := &storage.SessionContext{DomainSlug: slug, TestedConcepts: nil}
	newSess, err := c.store.CreateSession(completed.UserID, completed.DomainID, slug, nextKey, "explain", sctx)
	if err != nil {
		return nil, err
	}
	_ = c.store.UpsertProgress(storage.UserProgress{
		UserID:   completed.UserID,
		DomainID: completed.DomainID,
		NodeKey:  nextKey,
		Layer:    layer,
		Status:   "in_progress",
		Mastery:  0,
	})

	content, err := c.Begin(ctx, newSess)
	if err != nil {
		return nil, err
	}
	_, _ = c.store.AddMessage(newSess.ID, "assistant", content)

	intro := fmt.Sprintf("已进入下一节「%s」。\n\n%s", title, content)
	return &MessageResult{
		Role:          "assistant",
		Content:       intro,
		Phase:         "explain",
		NextSessionID: newSess.ID,
		NextNodeKey:   nextKey,
		NextNodeTitle: title,
	}, nil
}

func appendNextNodeHint(content string, tree *storage.KnowledgeTree, nodeKey string) string {
	content = strings.TrimSpace(content)
	nextKey, _, title, ok := domain.NextNodeAfter(tree, nodeKey)
	if !ok {
		if !strings.Contains(content, "全部完成") {
			content += "\n\n本课程节点已全部完成。"
		}
		return content
	}
	_ = nextKey
	hint := fmt.Sprintf("\n\n下一节：「%s」。回复「下一节」即可开始。", title)
	if !strings.Contains(content, "下一节") {
		content += hint
	}
	return content
}
