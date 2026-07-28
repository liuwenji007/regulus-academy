package domain

import (
	"fmt"
	"strings"
)

const (
	ContextWorkplace        = "workplace"
	ContextIntuition        = "intuition"
	ContextExamPattern      = "exam_pattern"
	ContextPrerequisiteLink = "prerequisite_link"

	DomainKindApplied   = "applied"
	DomainKindAcademic  = "academic"
	DomainKindMixed     = "mixed"

	ExerciseLevelRecognition = "recognition"
	ExerciseLevelRecall      = "recall"
	ExerciseLevelApply       = "apply"
)

// RequiresApplyExercise 入门层以概念讲解为主，不要求应用级练习。
func RequiresApplyExercise(layer string) bool {
	switch strings.TrimSpace(strings.ToLower(layer)) {
	case "入门", "entry":
		return false
	default:
		return true
	}
}

// NormalizeTeachingBeats 补齐 teaching_beats；无则按 core_concepts 生成 fallback。
func NormalizeTeachingBeats(spec *NodeSpec) []ConceptBeat {
	if spec == nil {
		return nil
	}
	if len(spec.TeachingBeats) > 0 {
		return spec.TeachingBeats
	}
	defaultCtx := defaultContextType(spec.DomainKind)
	beats := make([]ConceptBeat, 0, len(spec.CoreConcepts))
	for i, c := range spec.CoreConcepts {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		mustTeach := []string{c}
		if i < len(spec.CommonMistakes) && strings.TrimSpace(spec.CommonMistakes[i]) != "" {
			mustTeach = append(mustTeach, "常见误区："+strings.TrimSpace(spec.CommonMistakes[i]))
		}
		beats = append(beats, ConceptBeat{
			Concept:            c,
			MustTeach:          mustTeach,
			ContextType:        defaultCtx,
			FirstExerciseLevel: ExerciseLevelRecognition,
		})
	}
	return beats
}

func defaultContextType(domainKind string) string {
	switch strings.TrimSpace(strings.ToLower(domainKind)) {
	case DomainKindAcademic:
		return ContextIntuition
	case DomainKindMixed:
		return ContextIntuition
	default:
		return ContextWorkplace
	}
}

// EffectiveContextType 返回概念锚点类型。
func EffectiveContextType(beat ConceptBeat, spec *NodeSpec) string {
	if ct := strings.TrimSpace(strings.ToLower(beat.ContextType)); ct != "" {
		return ct
	}
	if spec != nil {
		return defaultContextType(spec.DomainKind)
	}
	return ContextWorkplace
}

// EffectiveFirstExerciseLevel 节点首题难度，默认 recognition。
func EffectiveFirstExerciseLevel(spec *NodeSpec) string {
	if spec == nil {
		return ExerciseLevelRecognition
	}
	if lv := strings.TrimSpace(strings.ToLower(spec.FirstExerciseLevel)); lv != "" {
		return lv
	}
	return ExerciseLevelRecognition
}

// MinMustTeachItems 每个 teaching_beat 至少一条非空 must_teach（有可教信号即可，不强制凑数）。
const MinMustTeachItems = 1

func countNonEmptyMustTeach(items []string) int {
	n := 0
	for _, s := range items {
		if strings.TrimSpace(s) != "" {
			n++
		}
	}
	return n
}

// EnsureConceptBeatMustTeachMin 保证每个 teaching_beat 至少有 min 条非空 must_teach（优化后复检不再爆量）。
func EnsureConceptBeatMustTeachMin(beats []ConceptBeat, min int, spec *NodeSpec) []ConceptBeat {
	if min <= 0 || len(beats) == 0 {
		return beats
	}
	out := make([]ConceptBeat, len(beats))
	for i, b := range beats {
		b.Concept = strings.TrimSpace(b.Concept)
		padded := make([]string, 0, len(b.MustTeach)+min)
		for _, s := range b.MustTeach {
			if strings.TrimSpace(s) != "" {
				padded = append(padded, s)
			}
		}
		if len(padded) >= min {
			b.MustTeach = padded
			out[i] = b
			continue
		}
		for len(padded) < min {
			added := false
			if spec != nil {
				idx := conceptIndexInCore(spec, b.Concept)
				if idx >= 0 && idx < len(spec.CommonMistakes) {
					cm := strings.TrimSpace(spec.CommonMistakes[idx])
					if cm != "" {
						hint := "常见误区：" + cm
						if !stringListContains(padded, hint) {
							padded = append(padded, hint)
							added = true
						}
					}
				}
			}
			if added {
				continue
			}
			concept := b.Concept
			if concept == "" {
				concept = "该概念"
			}
			fallback := fmt.Sprintf("掌握「%s」的定义、适用场景与识别方式", concept)
			if !stringListContains(padded, fallback) {
				padded = append(padded, fallback)
			} else {
				padded = append(padded, "补充该概念的关键教学要点")
			}
		}
		b.MustTeach = padded
		out[i] = b
	}
	return out
}

func conceptIndexInCore(spec *NodeSpec, concept string) int {
	if spec == nil {
		return -1
	}
	concept = strings.TrimSpace(concept)
	for i, c := range spec.CoreConcepts {
		if conceptMatchesTeaching(concept, c) {
			return i
		}
	}
	return -1
}

func stringListContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// BeatForConcept 按概念短语查找教学节拍。
func BeatForConcept(spec *NodeSpec, concept string) *ConceptBeat {
	if spec == nil {
		return nil
	}
	concept = strings.TrimSpace(concept)
	for _, b := range NormalizeTeachingBeats(spec) {
		if conceptMatchesTeaching(concept, b.Concept) {
			cp := b
			return &cp
		}
	}
	return nil
}

func conceptMatchesTeaching(a, b string) bool {
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

// ContextTypeLabel 供 Prompt 使用的中文锚点说明。
func ContextTypeLabel(ct string) string {
	switch strings.TrimSpace(strings.ToLower(ct)) {
	case ContextIntuition:
		return "直观理解/图景"
	case ContextExamPattern:
		return "典型题型/考法"
	case ContextPrerequisiteLink:
		return "与前后知识的衔接"
	default:
		return "工作/生产场景"
	}
}

// FormatTeachingBeatsForPrompt 格式化教学节拍注入 LLM 上下文。
func FormatTeachingBeatsForPrompt(spec *NodeSpec) string {
	if spec == nil {
		return ""
	}
	beats := NormalizeTeachingBeats(spec)
	if len(beats) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【教学节拍】\n")
	for _, beat := range beats {
		ct := EffectiveContextType(beat, spec)
		fmt.Fprintf(&b, "- %s（锚点：%s）\n", beat.Concept, ContextTypeLabel(ct))
		for _, line := range beat.MustTeach {
			line = strings.TrimSpace(line)
			if line != "" {
				fmt.Fprintf(&b, "  · %s\n", line)
			}
		}
	}
	if lv := EffectiveFirstExerciseLevel(spec); lv != "" {
		fmt.Fprintf(&b, "首题难度：%s\n", lv)
	}
	return strings.TrimSpace(b.String())
}

// UsesOverviewBegin 节点是否应在开场使用全景模式（仅定义、不三拍展开）。
func UsesOverviewBegin(spec *NodeSpec) bool {
	if spec == nil {
		return false
	}
	return len(spec.CoreConcepts) >= 3
}
