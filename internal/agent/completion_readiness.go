package agent

import (
	"context"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// LLMCompletionCheckEnabled 默认开启；设 REGULUS_LLM_COMPLETION_CHECK=0|false|no 回退纯规则点亮。
func LLMCompletionCheckEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_LLM_COMPLETION_CHECK")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// CompletionReadinessOpts 节点点亮前的综合评估入参（仅调用方传入字段）。
type CompletionReadinessOpts struct {
	UserMessage string
	SkipRequest bool // 用户主动申请掌握
}

type readinessEval struct {
	RuleDefer     DeferCompleteReason
	Uncovered     []string
	PriorFeedback string
}

func (c *Coach) evaluateCompletionReadiness(
	ctx context.Context,
	sess *storage.Session,
	sctx *storage.SessionContext,
	opts CompletionReadinessOpts,
	eval readinessEval,
) (MasteryCheckOutput, error) {
	schema, _ := domain.LoadSchema("mastery_check.json")
	in, err := c.buildInput(sess, completionReadinessInstruction(opts, eval), opts.UserMessage)
	if err != nil {
		return MasteryCheckOutput{}, err
	}
	in.RuleDeferReason = eval.RuleDefer
	in.ApplyExercisePassed = sctx != nil && sctx.ApplyExercisePassed
	in.UncoveredConcepts = append([]string{}, eval.Uncovered...)
	msgs := c.prompter.BuildMessages(in, TaskMasteryCheck, schema)
	ctx = observability.WithGeneration(ctx, TaskMasteryCheck.GenerationName())

	var out MasteryCheckOutput
	if err := c.llmClient(ctx).ChatJSON(ctx, msgs, 0.3, &out); err != nil {
		return MasteryCheckOutput{}, err
	}
	return out, nil
}

func completionReadinessInstruction(opts CompletionReadinessOpts, eval readinessEval) string {
	var b strings.Builder
	if opts.SkipRequest {
		b.WriteString("用户表示已掌握本节点、希望进入下一节。")
	} else {
		b.WriteString("用户刚答对当前练习；请结合全节对话与练习表现评估是否可点亮本节点。")
	}
	if fb := strings.TrimSpace(eval.PriorFeedback); fb != "" {
		b.WriteString("【本题批改反馈】")
		b.WriteString(fb)
		b.WriteString("。")
	}
	switch eval.RuleDefer {
	case DeferConceptCoverage:
		b.WriteString("系统规则建议：练习考查覆盖不足，通常应再练一题；详见上下文【规则待考查】。若答疑/深讲中已充分体现掌握，可 ready=true。")
	case DeferApplyExercise:
		b.WriteString("系统规则建议：尚未通过应用级练习，通常应再练一题。若对话中已有充分的代码或应用场景表现，可 ready=true。")
	default:
		b.WriteString("系统规则认为硬性练习门槛已满足，请做最终达标评估。")
	}
	b.WriteString("核心概念有明显缺口时 ready 应为 false；对在职开发者可适度从宽。")
	return b.String()
}

func mergeCompletionFeedback(parts ...string) string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "\n\n")
}

func (c *Coach) recordExerciseReinforcement(sess *storage.Session, sctx *storage.SessionContext) {
	if sctx == nil || sctx.Exercise == nil {
		return
	}
	for _, concept := range sctx.Exercise.ReinforcedConcepts {
		_ = c.store.IncrementReinforcement(sess.UserID, sess.DomainID, concept)
	}
}

func (c *Coach) tryCompleteAfterPass(
	ctx context.Context,
	sess *storage.Session,
	sctx *storage.SessionContext,
	priorFeedback string,
	core []string,
	layer string,
	opts CompletionReadinessOpts,
) (*MessageResult, error) {
	deferComplete, reason, uncovered := EvaluateDeferComplete(core, sctx.TestedConcepts, sctx, layer)
	eval := readinessEval{
		Uncovered:     uncovered,
		PriorFeedback: priorFeedback,
	}
	if deferComplete {
		eval.RuleDefer = reason
	}

	if !LLMCompletionCheckEnabled() {
		if opts.SkipRequest {
			return c.tryCompleteLegacySkip(ctx, sess, sctx, core, layer, opts, eval)
		}
		if deferComplete {
			return c.gradePassChainNextExercise(ctx, sess, sctx, priorFeedback, reason, uncovered)
		}
		c.recordExerciseReinforcement(sess, sctx)
		return c.completeNode(sess, sctx, priorFeedback)
	}

	readiness, err := c.evaluateCompletionReadiness(ctx, sess, sctx, opts, eval)
	if err != nil {
		if deferComplete {
			return c.gradePassChainNextExercise(ctx, sess, sctx, priorFeedback, reason, uncovered)
		}
		c.recordExerciseReinforcement(sess, sctx)
		return c.completeNode(sess, sctx, priorFeedback)
	}

	feedback := mergeCompletionFeedback(priorFeedback, readiness.Feedback)
	if readiness.Ready {
		c.recordExerciseReinforcement(sess, sctx)
		return c.completeNode(sess, sctx, feedback)
	}
	chainReason := resolveChainReason(eval.RuleDefer, sctx, layer)
	chainUncovered := uncovered
	if len(readiness.GapConcepts) > 0 {
		chainUncovered = mergeGapConcepts(readiness.GapConcepts, uncovered)
	}
	if feedback == "" {
		feedback = "还有一些薄弱点建议再巩固一下，你可以继续练习或补充说明。"
	}
	return c.deferAfterReadiness(ctx, sess, sctx, feedback, chainReason, chainUncovered, opts.SkipRequest)
}

// tryCompleteLegacySkip REGULUS_LLM_COMPLETION_CHECK=0 时申请掌握的评估路径（硬规则挡 + 不连题 review）。
func (c *Coach) tryCompleteLegacySkip(
	ctx context.Context,
	sess *storage.Session,
	sctx *storage.SessionContext,
	core []string,
	layer string,
	opts CompletionReadinessOpts,
	eval readinessEval,
) (*MessageResult, error) {
	readiness, err := c.evaluateCompletionReadiness(ctx, sess, sctx, opts, eval)
	if err != nil {
		return nil, err
	}
	if readiness.Ready {
		if deferComplete, reason, uncovered := EvaluateDeferComplete(core, sctx.TestedConcepts, sctx, layer); deferComplete {
			feedback := strings.TrimSpace(readiness.Feedback)
			if feedback == "" {
				feedback = "还有一些薄弱点建议再巩固一下，你可以继续练习或补充说明。"
			}
			return c.gradePassChainNextExercise(ctx, sess, sctx, feedback, reason, uncovered)
		}
		sctx.SkipMasteryWarned = false
		sctx.PendingSkipGaps = nil
		_ = storage.SaveSessionContext(sess, *sctx)
		return c.completeNode(sess, sctx, readiness.Feedback)
	}

	gaps := readiness.GapConcepts
	if len(gaps) == 0 && len(sctx.RecentMistakes) > 0 {
		gaps = append(gaps, sctx.RecentMistakes...)
	}
	sctx.SkipMasteryWarned = true
	sctx.PendingSkipGaps = gaps
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)

	feedback := strings.TrimSpace(readiness.Feedback)
	if feedback == "" {
		feedback = "还有一些薄弱点建议再巩固一下，你可以继续练习或补充说明。"
	}
	feedback += "\n\n若你确认当前水平已够用，可以再次说明「已经掌握，下一节」。"
	return &MessageResult{Role: "assistant", Content: feedback, Phase: sess.Phase}, nil
}

func resolveChainReason(ruleDefer DeferCompleteReason, sctx *storage.SessionContext, layer string) DeferCompleteReason {
	if ruleDefer != DeferNone {
		return ruleDefer
	}
	if sctx != nil && ApplyExerciseGateEnabled() && domain.RequiresApplyExercise(layer) && !sctx.ApplyExercisePassed {
		return DeferApplyExercise
	}
	return DeferConceptCoverage
}

func (c *Coach) deferAfterReadiness(
	ctx context.Context,
	sess *storage.Session,
	sctx *storage.SessionContext,
	feedback string,
	reason DeferCompleteReason,
	uncovered []string,
	skipRequest bool,
) (*MessageResult, error) {
	if skipRequest {
		sctx.SkipMasteryWarned = true
		sctx.PendingSkipGaps = append([]string{}, uncovered...)
		_ = storage.SaveSessionContext(sess, *sctx)
		_ = c.store.UpdateSession(sess)
		feedback += "\n\n若你确认当前水平已够用，可以再次说明「已经掌握，下一节」。"
	}
	return c.gradePassChainNextExercise(ctx, sess, sctx, feedback, reason, uncovered)
}
