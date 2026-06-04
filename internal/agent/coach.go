package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Coach 教学 Agent
type Coach struct {
	store    *storage.Store
	llm      atomic.Value // llm.Provider
	registry *domain.Registry
	prompter *Prompter
}

// NewCoach 创建 Coach
func NewCoach(store *storage.Store, llmClient llm.Provider) (*Coach, error) {
	p, err := NewPrompter()
	if err != nil {
		return nil, err
	}
	c := &Coach{
		store:    store,
		registry: domain.NewRegistry(),
		prompter: p,
	}
	c.llm.Store(llmClient)
	return c, nil
}

func (c *Coach) llmClient() llm.Provider {
	if v := c.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
}

// SetLLM 热更新 LLM 客户端（Web 修改模型配置后）
func (c *Coach) SetLLM(client llm.Provider) {
	if client != nil {
		c.llm.Store(client)
	}
}

// Begin 开场讲解
func (c *Coach) Begin(ctx context.Context, sess *storage.Session) (string, error) {
	if !c.llmClient().Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}
	in, err := c.buildInput(sess,
		"请做当前节点的开场讲解，并邀请用户提问或回复「开始练习」。",
		"")
	if err != nil {
		return "", err
	}
	msgs := c.prompter.BuildMessages(in, TaskBegin, "")
	content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return "", err
	}
	return content, nil
}

// HandleMessage 处理用户消息
func (c *Coach) HandleMessage(ctx context.Context, sess *storage.Session, userMsg string) (*MessageResult, error) {
	if !c.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	sctx := storage.ParseSessionContext(sess)

	if sess.Phase == "completed" {
		if wantsStartNext(userMsg) {
			return c.startNextNode(ctx, sess)
		}
		return c.completedQA(ctx, sess, &sctx, userMsg)
	}
	if wantsSkipMastery(userMsg) {
		return c.evaluateMasterySkip(ctx, sess, &sctx, userMsg)
	}
	if wantsStartNext(userMsg) {
		return c.blockStartNextUntilCompleted(sess), nil
	}

	switch sess.Phase {
	case "explain":
		if wantsExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		if wantsRealWorldCase(userMsg) {
			return c.realWorldCase(ctx, sess, &sctx)
		}
		return c.explainQA(ctx, sess, &sctx, userMsg)
	case "exercise":
		if wantsBackToExplain(userMsg) {
			sess.Phase = "explain"
			_ = c.store.UpdateSession(sess)
			return c.explainQA(ctx, sess, &sctx, userMsg)
		}
		if wantsNewExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		if wantsRealWorldCase(userMsg) {
			return c.realWorldCase(ctx, sess, &sctx)
		}
		return c.grade(ctx, sess, &sctx, userMsg)
	case "review":
		if wantsExercise(userMsg) || wantsNewExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		if wantsRealWorldCase(userMsg) {
			return c.realWorldCase(ctx, sess, &sctx)
		}
		return c.reviewExplain(ctx, sess, &sctx, userMsg)
	default:
		tree, _ := c.store.GetDomainTree(sess.UserID, sess.DomainID)
		hint := appendNextNodeHint("本节点已完成。", tree, sess.NodeKey)
		return &MessageResult{Role: "assistant", Content: hint, Phase: "completed"}, nil
	}
}

func (c *Coach) completedQA(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	in, err := c.buildInput(sess,
		"本节点已完成，用户仍有疑问。请仅针对本节点内容答疑，不要出新题；不要替用户进入下一节（用户需明确说「下一节」才会切换）。",
		userMsg)
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, TaskCompletedQA, "")
	content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
	if out, ok := parseExerciseJSONText(content); ok {
		return c.adoptExerciseOutput(sess, sctx, out)
	}
	content = sanitizeCoachPlainText(content)
	tree, _ := c.store.GetDomainTree(sess.UserID, sess.DomainID)
	content = appendNextNodeHint(content, tree, sess.NodeKey)
	return &MessageResult{Role: "assistant", Content: content, Phase: "completed"}, nil
}

func (c *Coach) explainQA(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	if wantsSkipMastery(userMsg) {
		return c.evaluateMasterySkip(ctx, sess, sctx, userMsg)
	}
	in, err := c.buildInput(sess,
		"请回答用户刚才的问题。不要自行宣称节点已通过或已点亮，那是 App 批改/申请完成流程的职责。",
		userMsg)
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, TaskExplainQA, "")
	content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
		if out, ok := parseExerciseJSONText(content); ok {
			return c.adoptExerciseOutput(sess, sctx, out)
		}
		content = sanitizeCoachPlainText(content)
		if looksLikeExerciseSubmitPrompt(content) {
			return c.adoptPlainTextExercise(sess, sctx, content)
		}
		return &MessageResult{Role: "assistant", Content: content, Phase: "explain"}, nil
}

func (c *Coach) realWorldCase(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext) (*MessageResult, error) {
	instruction := "请结合真实生产或工作场景，说明本节点概念如何落地：典型业务背景、关键代码片段或流程/架构设计（可精简但可对照）、为何这样设计并与概念对应。篇幅适中，用中文。"
	switch sess.Phase {
	case "exercise":
		instruction += "用户正在作答当前练习题，案例需帮助理解题意与概念，最后提醒可继续提交当前答案。"
	default:
		instruction += "最后邀请用户提问或回复「开始练习」。"
	}
	in, err := c.buildInput(sess, instruction, "")
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, TaskRealWorld, "")
	content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
	content = sanitizeCoachPlainText(content)
	res := &MessageResult{Role: "assistant", Content: content, Phase: sess.Phase}
	if sess.Phase == "exercise" {
		res.Exercise = exerciseMetaFromContext(sctx.Exercise)
	}
	return res, nil
}

func (c *Coach) startExercise(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext) (*MessageResult, error) {
	schema, _ := domain.LoadSchema("exercise.json")
	in, err := c.buildInput(sess, "请出一道针对当前节点的小练习。", "")
	if err != nil {
		return nil, err
	}
	reinforce := PickReinforceConcept(c.store, sess.UserID, sess.DomainID)
	in.Reinforce = reinforce
	msgs := c.prompter.BuildMessages(in, TaskExercise, schema)

	var out ExerciseOutput
	if err := c.llmClient().ChatJSON(ctx, msgs, 0.7, &out); err != nil {
		return nil, err
	}
	sctx.Exercise = BuildExerciseContext(out)
	sess.Phase = "exercise"
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)

	userContent := out.Question + "\n\n做完后直接把答案发给我。"
	return &MessageResult{
		Role:     "assistant",
		Content:  userContent,
		Phase:    "exercise",
		Exercise: exerciseMetaFromContext(sctx.Exercise),
	}, nil
}

func (c *Coach) grade(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, answer string) (*MessageResult, error) {
	schema, _ := domain.LoadSchema("grade.json")
	in, err := c.buildInput(sess, "请批改用户对当前题的作答。", answer)
	if err != nil {
		return nil, err
	}
	in.Exercise = sctx.Exercise
	msgs := c.prompter.BuildMessages(in, TaskGrade, schema)

	var out GradeOutput
	if err := c.llmClient().ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return nil, err
	}
	mergeGradeMistakes(&out)
	if fb, ok := parseGradeJSONText(out.Feedback); ok && strings.Contains(strings.TrimSpace(out.Feedback), "{") {
		out.Feedback = fb
	}
	if strings.TrimSpace(out.Feedback) == "" {
		out.Feedback = "这轮还没完全过关，建议再巩固一下。"
	}
	out.Feedback = sanitizeCoachPlainText(out.Feedback)

	res := &MessageResult{Role: "assistant", Content: out.Feedback, Phase: sess.Phase, ProgressUpdated: true}

	if out.Passed {
		if sctx.Exercise != nil {
			for _, concept := range sctx.Exercise.ReinforcedConcepts {
				_ = c.store.IncrementReinforcement(sess.UserID, sess.DomainID, concept)
			}
		}
		return c.completeNode(sess, sctx, out.Feedback)
	} else {
		sctx.Exercise = nil
		sctx.RecentMistakes = out.MistakeConcepts
		for _, concept := range out.MistakeConcepts {
			_ = c.store.UpsertMistake(sess.UserID, sess.DomainID, sess.NodeKey, concept)
		}
		sess.Phase = "review"
		res.Phase = "review"
		res.Exercise = nil
		if sctx.ReviewedOnce {
			res.Content = out.Feedback + "\n\n点击「再来一道」继续练习。"
		} else {
			sctx.ReviewedOnce = true
			res.Content = out.Feedback + "\n\n可以说「不懂，回讲解」，或点击「开始练习」再练一题。"
		}
	}
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)
	return res, nil
}

func (c *Coach) reviewExplain(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	if userMsg != "" && wantsSkipMastery(userMsg) {
		return c.evaluateMasterySkip(ctx, sess, sctx, userMsg)
	}
	turn := "请用更简单的方式讲清刚才薄弱的一点，并邀请用户回复「开始练习」。"
	if userMsg != "" {
		turn = userMsg
		in, err := c.buildInput(sess,
			"请回答用户刚才的问题。不要自行宣称节点已通过或已点亮，那是 App 批改/申请完成流程的职责。",
			userMsg)
		if err != nil {
			return nil, err
		}
		msgs := c.prompter.BuildMessages(in, TaskReview, "")
		content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
		if err != nil {
			return nil, err
		}
		if out, ok := parseExerciseJSONText(content); ok {
			return c.adoptExerciseOutput(sess, sctx, out)
		}
		content = sanitizeCoachPlainText(content)
		if looksLikeExerciseSubmitPrompt(content) {
			return c.adoptPlainTextExercise(sess, sctx, content)
		}
		return &MessageResult{Role: "assistant", Content: content, Phase: "review"}, nil
	}
	in, err := c.buildInput(sess, turn, "")
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, TaskReview, "")
	content, err := c.llmClient().ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
	content = sanitizeCoachPlainText(content)
	return &MessageResult{Role: "assistant", Content: content, Phase: "review"}, nil
}

func (c *Coach) buildInput(sess *storage.Session, taskInstruction, userMessage string) (PromptInput, error) {
	slug := sess.DomainSlug
	node, err := c.registry.GetNode(c.store, sess.DomainID, slug, sess.NodeKey)
	if err != nil {
		return PromptInput{}, err
	}
	tree, _ := c.store.GetDomainTree(sess.UserID, sess.DomainID)
	domainName := "Go 并发"
	if tree != nil {
		domainName = tree.DomainName
	}
	progress, _ := c.store.ListProgress(sess.UserID, sess.DomainID)
	sctx := storage.ParseSessionContext(sess)
	history, userToSend := c.loadChatHistory(sess.ID, userMessage)
	profile := ""
	if u, err := c.store.GetUser(sess.UserID); err == nil && u != nil {
		profile = u.ProfileSummary
	}
	var pendingPrereq []string
	if node != nil && len(node.Requires) > 0 {
		unmet := domain.UnmetRequireKeys(node.Requires, progress)
		for _, key := range unmet {
			title := key
			if tree != nil {
				title = domain.NodeTitle(tree, key)
			}
			pendingPrereq = append(pendingPrereq, title)
		}
	}
	return PromptInput{
		DomainName:          domainName,
		Node:                node,
		NodeKey:             sess.NodeKey,
		Layer:               node.Layer,
		Progress:            progress,
		Phase:               sess.Phase,
		TaskInstruction:     taskInstruction,
		UserMessage:         userToSend,
		Exercise:            sctx.Exercise,
		History:             history,
		RecentMistakes:      sctx.RecentMistakes,
		UserProfile:         profile,
		PendingPrereqTitles: pendingPrereq,
	}, nil
}

// loadChatHistory 加载会话历史；若最后一条用户消息与 userMessage 相同则不再重复追加
func (c *Coach) loadChatHistory(sessionID, userMessage string) ([]llm.Message, string) {
	msgs, err := c.store.ListMessages(sessionID)
	if err != nil {
		return nil, userMessage
	}
	history := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		history = append(history, llm.Message{Role: m.Role, Content: m.Content})
	}
	if userMessage != "" && len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" && last.Content == userMessage {
			return history, ""
		}
	}
	return history, userMessage
}
