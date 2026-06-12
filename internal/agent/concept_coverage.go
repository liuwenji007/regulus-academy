package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// DeferCompleteReason 延迟点亮节点的原因。
type DeferCompleteReason int

const (
	DeferNone DeferCompleteReason = iota
	DeferConceptCoverage
	DeferApplyExercise
)

// StrictConceptCoverageEnabled 默认开启；设 REGULUS_STRICT_CONCEPT_COVERAGE=0|false|no 可关闭混合完成门槛。
func StrictConceptCoverageEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_STRICT_CONCEPT_COVERAGE")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func conceptMatches(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

func conceptCovered(core string, tested []string) bool {
	for _, t := range tested {
		if conceptMatches(core, t) {
			return true
		}
	}
	return false
}

// UncoveredConcepts 返回 core 中尚未出现在 tested 里的概念（按 core 原文）。
func UncoveredConcepts(core, tested []string) []string {
	var out []string
	for _, c := range core {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !conceptCovered(c, tested) {
			out = append(out, c)
		}
	}
	return out
}

// ApplyExerciseGateEnabled 默认开启；设 REGULUS_REQUIRE_APPLY_EXERCISE=0|false|no 可关闭应用级练习门槛。
func ApplyExerciseGateEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_REQUIRE_APPLY_EXERCISE")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// EvaluateDeferComplete 判断答对后是否应继续练习再点亮。
func EvaluateDeferComplete(core, tested []string, sctx *storage.SessionContext, layer string) (deferComplete bool, reason DeferCompleteReason, uncovered []string) {
	uncovered = UncoveredConcepts(core, tested)
	if StrictConceptCoverageEnabled() && len(core) >= 3 && len(uncovered) >= 2 {
		return true, DeferConceptCoverage, uncovered
	}
	if ApplyExerciseGateEnabled() && domain.RequiresApplyExercise(layer) && sctx != nil && !sctx.ApplyExercisePassed {
		return true, DeferApplyExercise, uncovered
	}
	return false, DeferNone, uncovered
}

// MergeExplainedConcepts 将已深讲概念合并进会话，去重。
func MergeExplainedConcepts(sctx *storage.SessionContext, core, concepts []string) {
	if sctx == nil || len(concepts) == 0 {
		return
	}
	sctx.ExplainedConcepts = mergeConceptList(sctx.ExplainedConcepts, core, concepts)
}

func mergeConceptList(existing, core, add []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(add))
	out := make([]string, 0, len(existing)+len(add))
	appendOne := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		norm := NormalizeToCoreConcept(s, core)
		if norm == "" {
			norm = s
		}
		if _, ok := seen[norm]; ok {
			return
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	for _, e := range existing {
		appendOne(e)
	}
	for _, a := range add {
		appendOne(a)
	}
	return out
}

// EnsureExplainedConcepts 旧会话兼容：已有考查记录时，将已考概念视同已深讲。
func EnsureExplainedConcepts(sctx *storage.SessionContext, core []string) {
	if sctx == nil || len(core) == 0 {
		return
	}
	if len(sctx.ExplainedConcepts) > 0 {
		return
	}
	if len(sctx.TestedConcepts) == 0 {
		return
	}
	MergeExplainedConcepts(sctx, core, sctx.TestedConcepts)
}

// NormalizeToCoreConcept 将 reinforced 短语对齐到节点 core_concepts 条目（对齐失败则返回 trim 后的原串）。
func NormalizeToCoreConcept(reinforced string, core []string) string {
	r := strings.TrimSpace(reinforced)
	if r == "" {
		return ""
	}
	for _, c := range core {
		if conceptMatches(c, r) {
			return strings.TrimSpace(c)
		}
	}
	return r
}

// MergeTestedConcepts 将本题 reinforced 合并进 tested，去重。
func MergeTestedConcepts(tested, core, reinforced []string) []string {
	seen := make(map[string]struct{}, len(tested)+len(reinforced))
	out := make([]string, 0, len(tested)+len(reinforced))
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range tested {
		add(t)
	}
	for _, r := range reinforced {
		add(NormalizeToCoreConcept(r, core))
	}
	return out
}

// RecordExerciseTested 答对后把本题考查概念写入会话（未作答或答错不计入覆盖）。
func RecordExerciseTested(sctx *storage.SessionContext, core, reinforced []string) {
	if sctx == nil {
		return
	}
	sctx.TestedConcepts = MergeTestedConcepts(sctx.TestedConcepts, core, reinforced)
}

// FormatNextExerciseBridge 答对后自动连题时，在批改反馈与下一题之间的过渡句。
func FormatNextExerciseBridge(reason DeferCompleteReason, uncovered []string) string {
	if reason == DeferApplyExercise {
		return "接下来出一道应用级练习题（代码补全或找 bug）。"
	}
	if len(uncovered) == 0 {
		return "接下来再练一题。"
	}
	target := strings.TrimSpace(uncovered[0])
	if target == "" {
		return "接下来再练一题。"
	}
	return fmt.Sprintf("接下来考查：%s。", target)
}

// FormatDeferApplyNote 尚未通过应用级练习时的提示。
func FormatDeferApplyNote() string {
	return "\n\n本节点还需通过至少一道应用级练习（代码补全或找 bug）。"
}

// FormatDeferCompleteNote 覆盖率未达标时的简短进度提示（操作由界面按钮承接）。
func FormatDeferCompleteNote(uncovered []string) string {
	if len(uncovered) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"\n\n本节点还有 %d 个核心概念未在练习中考到：%s。",
		len(uncovered),
		strings.Join(uncovered, "；"),
	)
}

// exerciseTaskInstruction 动态出题任务说明（短句，原则性约束）。
func exerciseTaskInstruction(node *domain.NodeSpec, tested []string, explained []string, swap bool, requireApply bool) string {
	instr := "请出一道针对当前节点的小练习。"
	if requireApply {
		instr += "必须出一道 apply 级题：answer_format 为 json，question_type 为 code_fill 或 bug_find；结合工作场景要求写代码/补全/找 bug，禁止 choice 纯概念题。忽略 phase 中「首题 choice」的题序建议，本题必须为 apply 级。"
	}
	if node == nil || len(node.CoreConcepts) == 0 {
		if swap {
			instr += "与上一题考查概念尽量不同。"
		}
		return instr
	}
	if !requireApply {
		if len(tested) == 0 {
			instr += fmt.Sprintf("本会话首题，难度偏低（%s），可优先 choice 单概念识别。", domain.EffectiveFirstExerciseLevel(node))
		} else if len(tested) == 1 {
			instr += "第 2 题，可用 choice 或 short_answer。"
		} else {
			instr += "后续题可适当提升难度。"
		}
	}
	if uncovered := UncoveredConcepts(node.CoreConcepts, tested); len(uncovered) > 0 {
		instr += "优先考查待考查列表中的概念；不得考查对话历史（含开场讲解）中未出现过的概念。"
	}
	if len(explained) > 0 {
		instr += "reinforced_concepts 从本节点核心中选取，优先选已讲解过的。"
	}
	if swap {
		instr += "与上一题考查概念尽量不同。"
	}
	return instr
}
