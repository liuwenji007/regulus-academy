package agent

import (
	"context"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const noteDistillTimeout = 90 * time.Second

// scheduleNoteDistill 节点点亮后异步蒸馏对话为学习笔记，写入 node_notes 表
// 与 scheduleProfileRefresh 并发调用，互不依赖
func (c *Coach) scheduleNoteDistill(sess *storage.Session) {
	if sess == nil || c == nil {
		return
	}
	sessionID := sess.ID
	userID := sess.UserID
	domainID := sess.DomainID
	nodeKey := sess.NodeKey

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), noteDistillTimeout)
		defer cancel()
		ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
			Name:      "coach.note_distill",
			UserID:    userID,
			SessionID: sessionID,
		})
		defer endTrace()

		current, err := c.store.GetSession(sessionID)
		if err != nil || current == nil || current.UserID != userID {
			return
		}
		_ = c.DistillNodeNote(ctx, current, domainID, nodeKey)
	}()
}

// DistillNodeNote 读取本节对话 + 错题，调用 LLM 生成学习笔记，写入 node_notes 表
func (c *Coach) DistillNodeNote(ctx context.Context, sess *storage.Session, domainID, nodeKey string) error {
	if c == nil || sess == nil || !c.llmClient(ctx).Configured() {
		return nil
	}

	msgs, err := c.store.ListMessages(sess.ID)
	if err != nil {
		return err
	}
	transcript := formatTranscriptForProfile(msgs)
	if strings.TrimSpace(transcript) == "" {
		return nil
	}

	// 读取本节错题
	mistakes, _ := c.store.ListMistakesByNode(sess.UserID, domainID)
	nodeMistakes := mistakes[nodeKey]

	// 读取节点边界（含 core_concepts）
	node, err := c.registry.GetNode(c.store, domainID, sess.DomainSlug, nodeKey)
	if err != nil {
		return err
	}

	// 读取领域名
	domainName := "课程"
	if tree, tErr := c.store.GetDomainTree(sess.UserID, domainID); tErr == nil && tree != nil {
		domainName = tree.DomainName
	}

	// 组装 TaskInstruction 作为额外上下文
	var instrParts []string
	instrParts = append(instrParts, "请根据以下本节对话与信息，为学生生成一篇学习笔记。")
	instrParts = append(instrParts, "【节点】："+node.Node+"（"+domainName+" · "+node.Layer+"）")
	if len(node.CoreConcepts) > 0 {
		instrParts = append(instrParts, "【核心概念】："+strings.Join(node.CoreConcepts, "、"))
	}
	if len(nodeMistakes) > 0 {
		instrParts = append(instrParts, "【错题记录】："+strings.Join(nodeMistakes, "；"))
	}

	in := PromptInput{
		DomainName:      domainName,
		Node:            node,
		NodeKey:         nodeKey,
		Layer:           node.Layer,
		Phase:           "completed",
		TaskInstruction: strings.Join(instrParts, "\n"),
		UserMessage:     "【本节对话摘录】\n" + transcript,
	}

	llmMsgs := c.prompter.BuildMessages(in, TaskNoteDistill, "")
	ctx = observability.WithGeneration(ctx, TaskNoteDistill.GenerationName())

	result, err := c.llmClient(ctx).ChatWithTemp(ctx, llmMsgs, 0.5)
	if err != nil {
		return err
	}
	content := strings.TrimSpace(result)
	if content == "" {
		return nil
	}

	return c.store.UpsertNodeNote(sess.UserID, domainID, nodeKey, content)
}
