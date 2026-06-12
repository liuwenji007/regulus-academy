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

	var core []string
	layer := ""
	if node, err := c.registry.GetNode(c.store, sess.DomainID, sess.DomainSlug, sess.NodeKey); err == nil && node != nil {
		core = node.CoreConcepts
		layer = node.Layer
	}

	return c.tryCompleteAfterPass(ctx, sess, sctx, "", core, layer, CompletionReadinessOpts{
		UserMessage: userMsg,
		SkipRequest: true,
	})
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

	progress, _ := c.store.ListProgress(sess.UserID, sess.DomainID)
	completedKeys := domain.CompletedKeysFromProgress(progress)
	content := appendNextNodeHint(strings.TrimSpace(feedback), tree, sess.NodeKey, completedKeys)
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
		if nextKey, _, nextTitle, ok := domain.NextUncompletedNodeAfter(tree, sess.NodeKey, completedKeys); ok {
			res.NextNodeKey = nextKey
			res.NextNodeTitle = nextTitle
		}
	}
	c.scheduleProfileRefresh(sess, sctx)
	c.scheduleNoteDistill(sess)
	return res, nil
}
