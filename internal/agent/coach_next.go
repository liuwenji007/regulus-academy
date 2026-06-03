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
	sctx := &storage.SessionContext{DomainSlug: slug}
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
