package domain

import (
	"encoding/json"
	"strings"
)

// ConceptBeat 单个核心概念的教学节拍
type ConceptBeat struct {
	Concept            string   `yaml:"concept" json:"concept"`
	MustTeach          []string `yaml:"must_teach" json:"must_teach"`
	ContextType        string   `yaml:"context_type,omitempty" json:"context_type,omitempty"`
	FirstExerciseLevel string   `yaml:"first_exercise_level,omitempty" json:"first_exercise_level,omitempty"`
}

// UnmarshalJSON 容错解析：LLM 偶尔把 teaching_beats 元素输出为纯字符串、
// 或把 must_teach 输出为单个字符串，这里都按等价对象接受。
func (b *ConceptBeat) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		*b = ConceptBeat{Concept: strings.TrimSpace(plain)}
		return nil
	}
	var aux struct {
		Concept            string          `json:"concept"`
		MustTeach          json.RawMessage `json:"must_teach"`
		ContextType        string          `json:"context_type"`
		FirstExerciseLevel string          `json:"first_exercise_level"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*b = ConceptBeat{
		Concept:            aux.Concept,
		ContextType:        aux.ContextType,
		FirstExerciseLevel: aux.FirstExerciseLevel,
	}
	if len(aux.MustTeach) > 0 {
		var list []string
		if err := json.Unmarshal(aux.MustTeach, &list); err == nil {
			b.MustTeach = list
		} else {
			var one string
			if err := json.Unmarshal(aux.MustTeach, &one); err == nil && strings.TrimSpace(one) != "" {
				b.MustTeach = []string{one}
			}
		}
	}
	return nil
}

// NodeSpec 节点边界定义（来自 nodes/*.yaml 或 LLM 生成）
type NodeSpec struct {
	Node           string   `yaml:"node" json:"node"`
	Key            string   `yaml:"key" json:"key"`
	Layer          string   `yaml:"layer" json:"layer"`
	Requires       []string `yaml:"requires,omitempty" json:"requires,omitempty"`
	CoreConcepts   []string `yaml:"core_concepts" json:"core_concepts"`
	CommonMistakes []string `yaml:"common_mistakes" json:"common_mistakes"`
	Boundaries     []string `yaml:"boundaries" json:"boundaries"`
	ExerciseIdeas  []string `yaml:"exercise_ideas" json:"exercise_ideas"`
	GradingHints   []string `yaml:"grading_hints,omitempty" json:"grading_hints,omitempty"`
	TeachingBeats      []ConceptBeat `yaml:"teaching_beats,omitempty" json:"teaching_beats,omitempty"`
	FirstExerciseLevel string        `yaml:"first_exercise_level,omitempty" json:"first_exercise_level,omitempty"`
	DomainKind         string        `yaml:"domain_kind,omitempty" json:"domain_kind,omitempty"`
}

// TreeFile tree.yaml 结构
type TreeFile struct {
	Domain      string                  `yaml:"domain" json:"domain"`
	Slug        string                  `yaml:"slug" json:"slug"`
	ParentSlug  string                  `yaml:"parent_slug" json:"parentSlug,omitempty"`
	Version     int                     `yaml:"version" json:"version"`
	Description string                  `yaml:"description" json:"description"`
	Modules     []TreeModuleDef         `yaml:"modules,omitempty" json:"modules,omitempty"`
	Layers      map[string]TreeLayerDef `yaml:"layers" json:"layers"`
}

// TreeModuleDef 主题模块定义
type TreeModuleDef struct {
	Key   string   `yaml:"key" json:"key"`
	Label string   `yaml:"label" json:"label"`
	Goal  string   `yaml:"goal,omitempty" json:"goal,omitempty"`
	Order int      `yaml:"order,omitempty" json:"order,omitempty"`
	Nodes []string `yaml:"nodes" json:"nodes"`
}

// TreeLayerDef 层级定义
type TreeLayerDef struct {
	Label string        `yaml:"label" json:"label"`
	Time  string        `yaml:"time" json:"time"`
	Goal  string        `yaml:"goal" json:"goal"`
	Nodes []TreeNodeDef `yaml:"nodes" json:"nodes"`
}

// TreeNodeDef 节点引用
type TreeNodeDef struct {
	Key      string   `yaml:"key" json:"key"`
	Title    string   `yaml:"title" json:"title"`
	Requires []string `yaml:"requires,omitempty" json:"requires,omitempty"`
}
