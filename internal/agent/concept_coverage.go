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

// FormatDeferCompleteNote 批改/掌握度通过但被覆盖率拦截时的用户可见补充说明。
func FormatDeferCompleteNote(uncovered []string) string {
	if len(uncovered) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"\n\n本节点还有 %d 个核心概念未在练习中考到，建议再来一道：%s。",
		len(uncovered),
		strings.Join(uncovered, "；"),
	)
}

// exerciseTaskInstruction 动态出题任务说明（短句，避免堆禁令）。
func exerciseTaskInstruction(node *domain.NodeSpec, tested []string, swap bool) string {
	instr := "请出一道针对当前节点的小练习。"
	if node == nil || len(node.CoreConcepts) == 0 {
		if swap {
			instr += "与上一题考查概念尽量不同。"
		}
		return instr
	}
	if uncovered := UncoveredConcepts(node.CoreConcepts, tested); len(uncovered) > 0 {
		instr += "优先考查待覆盖列表中的概念；reinforced_concepts 从【本节点】核心中选取并填写。"
	}
	if swap {
		instr += "与上一题考查概念尽量不同。"
	}
	return instr
}
