package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Coach 教学 Agent
type Coach struct {
	store    *storage.Store
	llm      *llm.Client
	registry *domain.Registry
	prompter *Prompter
}

// NewCoach 创建 Coach
func NewCoach(store *storage.Store, llmClient *llm.Client) (*Coach, error) {
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
		return "", fmt.Errorf("未配置 DEEPSEEK_API_KEY")
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
		return nil, fmt.Errorf("未配置 DEEPSEEK_API_KEY")
	}
	sctx := storage.ParseSessionContext(sess)

	switch sess.Phase {
	case "explain":
		if wantsExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
		}
		return c.explainQA(ctx, sess, &sctx, userMsg)
	case "exercise":
		return c.grade(ctx, sess, &sctx, userMsg)
	case "review":
		if wantsExercise(userMsg) {
			return c.startExercise(ctx, sess, &sctx)
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
	sctx.Exercise = &storage.ExerciseContext{
		Question:           out.Question,
		QuestionType:       out.QuestionType,
		ReinforcedConcepts: out.ReinforcedConcepts,
	}
	sess.Phase = "exercise"
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)

	userContent := out.Question + "\n\n做完后直接把答案发给我。"
	return &MessageResult{Role: "assistant", Content: userContent, Phase: "exercise"}, nil
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
		sess.Phase = "completed"
		sess.Status = "completed"
		tree, _ := c.store.GetDomainTree(sess.DomainID)
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
		if sctx.Exercise != nil {
			for _, concept := range sctx.Exercise.ReinforcedConcepts {
				_ = c.store.IncrementReinforcement(sess.UserID, sess.DomainID, concept)
			}
		}
		res.Phase = "completed"
		res.NodeCompleted = true
		res.Content = out.Feedback + "\n\n节点已点亮，返回知识树继续下一节吧。"
	} else {
		for _, concept := range out.MistakeConcepts {
			_ = c.store.UpsertMistake(sess.UserID, sess.DomainID, sess.NodeKey, concept)
		}
		if sctx.ReviewedOnce {
			sess.Phase = "review"
			res.Content = out.Feedback + "\n\n回复「开始练习」可以再试一题。"
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
	if slug == "" {
		slug = "go-concurrency"
	}
	node, err := c.registry.LoadNode(slug, sess.NodeKey)
	if err != nil {
		return PromptInput{}, err
	}
	tree, _ := c.store.GetDomainTree(sess.DomainID)
	domainName := "Go 并发"
	if tree != nil {
		domainName = tree.DomainName
	}
	progress, _ := c.store.ListProgress(sess.UserID, sess.DomainID)
	sctx := storage.ParseSessionContext(sess)
	return PromptInput{
		DomainName: domainName,
		Node:       node,
		Layer:      node.Layer,
		Progress:   progress,
		Phase:      sess.Phase,
		Turn:       turn,
		Exercise:   sctx.Exercise,
	}, nil
}

func wantsExercise(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	keywords := []string{"开始练习", "准备好了", "出题", "开始", "练习"}
	for _, k := range keywords {
		if strings.Contains(m, k) {
			return true
		}
	}
	return false
}
