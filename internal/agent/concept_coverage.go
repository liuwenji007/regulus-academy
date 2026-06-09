package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
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

// ShouldDeferComplete hybrid：core≥3 且未覆盖≥2 时建议再练一题再点亮。
func ShouldDeferComplete(core, tested []string) (shouldDefer bool, uncovered []string) {
	if !StrictConceptCoverageEnabled() {
		return false, nil
	}
	uncovered = UncoveredConcepts(core, tested)
	return len(core) >= 3 && len(uncovered) >= 2, uncovered
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

// ConceptDeepExplained 概念是否已深讲过。
func ConceptDeepExplained(concept string, explained []string) bool {
	return conceptCovered(concept, explained)
}

// NextExerciseTargetConcept 下一题应考查的概念（优先未考）。
func NextExerciseTargetConcept(core, tested []string) string {
	if uncovered := UncoveredConcepts(core, tested); len(uncovered) > 0 {
		return uncovered[0]
	}
	if len(core) > 0 {
		return strings.TrimSpace(core[0])
	}
	return ""
}

// NextConceptToDeepen 练前/练后选择下一个需深讲的概念。
func NextConceptToDeepen(core, explained, tested []string, afterPass bool) string {
	if afterPass {
		if uncovered := UncoveredConcepts(core, tested); len(uncovered) > 0 {
			return uncovered[0]
		}
		return ""
	}
	for _, c := range core {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !ConceptDeepExplained(c, explained) {
			return c
		}
	}
	return NextExerciseTargetConcept(core, tested)
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
func exerciseTaskInstruction(node *domain.NodeSpec, tested []string, explained []string, swap bool) string {
	instr := "请出一道针对当前节点的小练习。"
	if node == nil || len(node.CoreConcepts) == 0 {
		if swap {
			instr += "与上一题考查概念尽量不同。"
		}
		return instr
	}
	if len(tested) == 0 {
		instr += fmt.Sprintf("本会话首题，难度偏低（%s），可优先 choice 单概念识别。", domain.EffectiveFirstExerciseLevel(node))
	} else if len(tested) == 1 {
		instr += "第 2 题，可用 choice 或 short_answer。"
	} else {
		instr += "后续题可适当提升难度。"
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
