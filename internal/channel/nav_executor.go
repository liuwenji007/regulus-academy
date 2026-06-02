package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (r *Router) executeNavigation(ctx context.Context, userID string, intent NavigationIntent, rawText string) HandleResult {
	switch intent.Action {
	case NavHelp:
		return HandleResult{Replies: []string{helpText()}}
	case NavListCourses:
		return HandleResult{Replies: r.handleCourses(userID)}
	case NavProgress:
		return HandleResult{Replies: r.handleProgress(userID)}
	case NavContinue:
		if r.shouldContinueToCoach(userID) {
			return r.handleChat(ctx, userID, rawText)
		}
		return HandleResult{Replies: r.handleContinue(ctx, userID)}
	case NavShowNodes:
		ref := intent.CourseRef
		if ref == "" && intent.NodeRef != "" {
			ref = intent.NodeRef
		}
		return HandleResult{Replies: r.handleLearn(userID, ref)}
	case NavStartNode:
		return r.executeStartNode(ctx, userID, intent)
	case NavClarify:
		msg := strings.TrimSpace(intent.ReplyHint)
		if msg == "" {
			msg = "请说明你想查看哪门课程或哪个节点，也可以说「我的课程」查看列表。"
		}
		return HandleResult{Replies: []string{msg}}
	default:
		return HandleResult{Replies: []string{"暂时无法理解，请说「帮助」查看可用操作。"}}
	}
}

func (r *Router) shouldContinueToCoach(userID string) bool {
	sess, err := r.sessions.ActiveSessionForUser(userID)
	return err == nil && sess != nil
}

func (r *Router) executeStartNode(ctx context.Context, userID string, intent NavigationIntent) HandleResult {
	list, err := r.store.ListDomainSummaries(userID)
	if err != nil || len(list) == 0 {
		return HandleResult{Replies: []string{"还没有课程。请先在 Web 端创建知识库。"}}
	}

	domainID := ""
	if intent.CourseRef != "" {
		domainID, _ = resolveCourseRef(list, intent.CourseRef)
	}
	if domainID == "" {
		r.mu.Lock()
		domainID = r.pending[userID]
		r.mu.Unlock()
	}
	if domainID == "" {
		active, _ := r.store.GetChannelActiveNode(userID)
		if active != nil {
			domainID = active.DomainID
		}
	}
	if domainID == "" && len(list) == 1 {
		domainID = list[0].ID
	}
	if domainID == "" {
		return HandleResult{Replies: []string{"请先说明要学哪门课，或发送「我的课程」查看列表。"}}
	}

	if intent.CourseRef != "" {
		r.mu.Lock()
		r.pending[userID] = domainID
		r.mu.Unlock()
	}

	tree, err := r.store.GetDomainTree(userID, domainID)
	if err != nil || tree == nil {
		return HandleResult{Replies: []string{"无法加载知识树。"}}
	}
	nodes := flattenNodes(tree)
	nodeRef := intent.NodeRef
	if nodeRef == "" {
		nodeRef = "1"
	}
	nodeKey, layer, ok := resolveNodeRef(nodes, nodeRef)
	if !ok {
		return HandleResult{Replies: []string{"未找到该节点。请先说课程名查看节点列表，或指定「第 N 个节点」。"}}
	}
	return HandleResult{Replies: r.handleNodeByKey(ctx, userID, domainID, nodeKey, layer, tree)}
}

func (r *Router) handleNodeByKey(ctx context.Context, userID, domainID, nodeKey, layer string, tree *storage.KnowledgeTree) []string {
	result, err := r.sessions.StartOrResumeSession(ctx, userID, domainID, nodeKey, layer)
	if err != nil {
		return []string{"开始学习失败：" + err.Error()}
	}
	_ = r.store.SetChannelActiveNode(userID, domainID, nodeKey)

	if result.Resumed {
		msgs, _ := r.store.ListMessages(result.Session.ID)
		last := ""
		if len(msgs) > 0 {
			last = msgs[len(msgs)-1].Content
		}
		title := domain.NodeTitle(tree, nodeKey)
		if last != "" {
			return []string{fmt.Sprintf("继续学习「%s」（阶段：%s）。上一条：\n%s\n\n直接回复即可继续。", title, result.Session.Phase, truncate(last, 200))}
		}
		return []string{fmt.Sprintf("继续学习「%s」（阶段：%s）。直接回复即可。", title, result.Session.Phase)}
	}
	title := domain.NodeTitle(tree, nodeKey)
	return []string{fmt.Sprintf("开始学习「%s」\n\n%s", title, result.Content)}
}
