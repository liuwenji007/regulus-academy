package agent

import (
	"context"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (c *Coach) evaluateMasterySkip(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	if sctx.SkipMasteryWarned {
		return c.forceCompleteWithGapRecording(sess, sctx)
	}

	schema, _ := domain.LoadSchema("mastery_check.json")
	in, err := c.buildInput(sess, "用户表示已掌握本节点、希望进入下一节。请根据对话历史、练习与答疑表现评估是否达到本节点学习目标。对在职开发者可适度从宽，但核心概念有明显缺口时不应放行。")
	if err != nil {
		return nil, err
	}
	in.Turn = userMsg
	msgs := c.prompter.BuildMessages(in, schema)

	var out MasteryCheckOutput
	if err := c.llm.ChatJSON(ctx, msgs, 0.3, &out); err != nil {
		return nil, err
	}

	if out.Ready {
		sctx.SkipMasteryWarned = false
		sctx.PendingSkipGaps = nil
		_ = storage.SaveSessionContext(sess, *sctx)
		return c.completeNode(sess, sctx, out.Feedback)
	}

	gaps := out.GapConcepts
	if len(gaps) == 0 && len(sctx.RecentMistakes) > 0 {
		gaps = append(gaps, sctx.RecentMistakes...)
	}
	sctx.SkipMasteryWarned = true
	sctx.PendingSkipGaps = gaps
	_ = storage.SaveSessionContext(sess, *sctx)

	feedback := strings.TrimSpace(out.Feedback)
	if feedback == "" {
		feedback = "还有一些薄弱点建议再巩固一下，你可以继续练习或补充说明。"
	}
	feedback += "\n\n若你确认当前水平已够用，可以再次说明「已经掌握，下一节」。"
	return &MessageResult{Role: "assistant", Content: feedback, Phase: sess.Phase}, nil
}

func (c *Coach) forceCompleteWithGapRecording(sess *storage.Session, sctx *storage.SessionContext) (*MessageResult, error) {
	gaps := sctx.PendingSkipGaps
	if len(gaps) == 0 {
		gaps = sctx.RecentMistakes
	}
	for _, concept := range gaps {
		concept = strings.TrimSpace(concept)
		if concept == "" {
			continue
		}
		_ = c.store.UpsertMistake(sess.UserID, sess.DomainID, sess.NodeKey, concept)
	}
	sctx.SkipMasteryWarned = false
	sctx.PendingSkipGaps = nil
	sctx.Exercise = nil
	_ = storage.SaveSessionContext(sess, *sctx)
	return c.completeNode(sess, sctx, "好的，本节点已为你标记完成。")
}

func (c *Coach) completeNode(sess *storage.Session, sctx *storage.SessionContext, feedback string) (*MessageResult, error) {
	if sctx != nil {
		sctx.Exercise = nil
		sctx.SkipMasteryWarned = false
		sctx.PendingSkipGaps = nil
		_ = storage.SaveSessionContext(sess, *sctx)
	}
	sess.Phase = "completed"
	sess.Status = "completed"
	tree, _ := c.store.GetDomainTree(sess.UserID, sess.DomainID)
	layer := "entry"
	if tree != nil {
		layer = domain.LayerForNode(tree, sess.NodeKey)
	}
	_ = c.store.UpsertProgress(storage.UserProgress{
		UserID:   sess.UserID,
		DomainID: sess.DomainID,
		NodeKey:  sess.NodeKey,
		Layer:    layer,
		Status:   "completed",
		Mastery:  0.8,
	})
	_ = c.store.UpdateSession(sess)

	content := appendNextNodeHint(strings.TrimSpace(feedback), tree, sess.NodeKey)
	if !strings.Contains(content, "节点已点亮") && !strings.Contains(content, "下一节") {
		content = strings.TrimSpace(content) + "\n\n节点已点亮。"
	}
	res := &MessageResult{
		Role:            "assistant",
		Content:         content,
		Phase:           "completed",
		NodeCompleted:   true,
		ProgressUpdated: true,
	}
	if tree != nil {
		if nextKey, _, nextTitle, ok := domain.NextNodeAfter(tree, sess.NodeKey); ok {
			res.NextNodeKey = nextKey
			res.NextNodeTitle = nextTitle
		}
	}
	return res, nil
}
