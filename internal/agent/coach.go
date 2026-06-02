package agent

import (
	"context"
	"fmt"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Coach 教学 Agent
type Coach struct {
	store    *storage.Store
	llm      llm.Provider
	registry *domain.Registry
	prompter *Prompter
}

// NewCoach 创建 Coach
func NewCoach(store *storage.Store, llmClient llm.Provider) (*Coach, error) {
	p, err := NewPrompter()
	if err != nil {
		return nil, err
	}
	return &Coach{
		store:    store,
		llm:      llmClient,
		registry: domain.NewRegistry(),
		prompter: p,
	}, nil
}

// Begin 开场讲解
func (c *Coach) Begin(ctx context.Context, sess *storage.Session) (string, error) {
	if !c.llm.Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}
	in, err := c.buildInput(sess, "请做当前节点的开场讲解，并邀请用户提问或回复「开始练习」。")
	if err != nil {
		return "", err
	}
	msgs := c.prompter.BuildMessages(in, "")
	content, err := c.llm.ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return "", err
	}
	return content, nil
}

// HandleMessage 处理用户消息
func (c *Coach) HandleMessage(ctx context.Context, sess *storage.Session, userMsg string) (*MessageResult, error) {
	if !c.llm.Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	sctx := storage.ParseSessionContext(sess)

	switch sess.Phase {
	case "explain":
		if wantsExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		if wantsRealWorldCase(userMsg) {
			return c.realWorldCase(ctx, sess, &sctx)
		}
		if wantsSkipMastery(userMsg) {
			return c.evaluateMasterySkip(ctx, sess, &sctx, userMsg)
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
		if wantsSkipMastery(userMsg) {
			return c.evaluateMasterySkip(ctx, sess, &sctx, userMsg)
		}
		return c.grade(ctx, sess, &sctx, userMsg)
	case "review":
		if wantsExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		if wantsRealWorldCase(userMsg) {
			return c.realWorldCase(ctx, sess, &sctx)
		}
		if wantsSkipMastery(userMsg) {
			return c.evaluateMasterySkip(ctx, sess, &sctx, userMsg)
		}
		return c.reviewExplain(ctx, sess, &sctx, userMsg)
	default:
		return &MessageResult{Role: "assistant", Content: "本节点已完成，返回知识树选择下一个节点吧。", Phase: sess.Phase}, nil
	}
}

func (c *Coach) explainQA(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	in, err := c.buildInput(sess, "请回答用户刚才的问题。")
	if err != nil {
		return nil, err
	}
	in.Turn = userMsg
	msgs := c.prompter.BuildMessages(in, "")
	content, err := c.llm.ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
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
	in, err := c.buildInput(sess, instruction)
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, "")
	content, err := c.llm.ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
	res := &MessageResult{Role: "assistant", Content: content, Phase: sess.Phase}
	if sess.Phase == "exercise" {
		res.Exercise = exerciseMetaFromContext(sctx.Exercise)
	}
	return res, nil
}

func (c *Coach) startExercise(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext) (*MessageResult, error) {
	schema, _ := domain.LoadSchema("exercise.json")
	in, err := c.buildInput(sess, "请出一道针对当前节点的小练习。")
	if err != nil {
		return nil, err
	}
	reinforce := PickReinforceConcept(c.store, sess.UserID, sess.DomainID)
	in.Reinforce = reinforce
	msgs := c.prompter.BuildMessages(in, schema)

	var out ExerciseOutput
	if err := c.llm.ChatJSON(ctx, msgs, 0.7, &out); err != nil {
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
	in, err := c.buildInput(sess, "请批改用户对当前题的作答。")
	if err != nil {
		return nil, err
	}
	in.Turn = answer
	in.Exercise = sctx.Exercise
	msgs := c.prompter.BuildMessages(in, schema)

	var out GradeOutput
	if err := c.llm.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return nil, err
	}

	res := &MessageResult{Role: "assistant", Content: out.Feedback, Phase: sess.Phase, ProgressUpdated: true}

	if out.Passed {
		if sctx.Exercise != nil {
			for _, concept := range sctx.Exercise.ReinforcedConcepts {
				_ = c.store.IncrementReinforcement(sess.UserID, sess.DomainID, concept)
			}
		}
		return c.completeNode(sess, sctx, out.Feedback)
	} else {
		sctx.RecentMistakes = out.MistakeConcepts
		for _, concept := range out.MistakeConcepts {
			_ = c.store.UpsertMistake(sess.UserID, sess.DomainID, sess.NodeKey, concept)
		}
		if sctx.ReviewedOnce {
			sess.Phase = "review"
			res.Phase = "review"
			res.Content = out.Feedback + "\n\n点击「再来一道」继续练习。"
		} else {
			sctx.ReviewedOnce = true
			sess.Phase = "review"
			review, err := c.reviewExplain(ctx, sess, sctx, "")
			if err != nil {
				res.Phase = "review"
				res.Content = out.Feedback
			} else {
				res.Content = out.Feedback + "\n\n" + review.Content
				res.Phase = review.Phase
			}
		}
	}
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)
	return res, nil
}

func (c *Coach) reviewExplain(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext, userMsg string) (*MessageResult, error) {
	turn := "请用更简单的方式讲清刚才薄弱的一点，并邀请用户回复「开始练习」。"
	if userMsg != "" {
		turn = userMsg
		in, err := c.buildInput(sess, "请回答用户刚才的问题。")
		if err != nil {
			return nil, err
		}
		in.Turn = userMsg
		msgs := c.prompter.BuildMessages(in, "")
		content, err := c.llm.ChatWithTemp(ctx, msgs, 0.6)
		if err != nil {
			return nil, err
		}
		return &MessageResult{Role: "assistant", Content: content, Phase: "review"}, nil
	}
	in, err := c.buildInput(sess, turn)
	if err != nil {
		return nil, err
	}
	msgs := c.prompter.BuildMessages(in, "")
	content, err := c.llm.ChatWithTemp(ctx, msgs, 0.6)
	if err != nil {
		return nil, err
	}
	return &MessageResult{Role: "assistant", Content: content, Phase: "review"}, nil
}

func (c *Coach) buildInput(sess *storage.Session, turn string) (PromptInput, error) {
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
	history, turnToSend := c.loadChatHistory(sess.ID, turn)
	profile := ""
	if u, err := c.store.GetUser(sess.UserID); err == nil && u != nil {
		profile = u.ProfileSummary
	}
	return PromptInput{
		DomainName:     domainName,
		Node:           node,
		Layer:          node.Layer,
		Progress:       progress,
		Phase:          sess.Phase,
		Turn:           turnToSend,
		Exercise:       sctx.Exercise,
		History:        history,
		RecentMistakes: sctx.RecentMistakes,
		UserProfile:    profile,
	}, nil
}

// loadChatHistory 加载会话历史；若最后一条用户消息与 turn 相同则不再重复追加
func (c *Coach) loadChatHistory(sessionID, turn string) ([]llm.Message, string) {
	msgs, err := c.store.ListMessages(sessionID)
	if err != nil {
		return nil, turn
	}
	history := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		history = append(history, llm.Message{Role: m.Role, Content: m.Content})
	}
	if turn != "" && len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" && last.Content == turn {
			return history, ""
		}
	}
	return history, turn
}
