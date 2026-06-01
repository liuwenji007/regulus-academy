package domain

// NodeSpec 节点边界定义（来自 nodes/*.yaml 或 LLM 生成）
type NodeSpec struct {
	Node           string   `yaml:"node" json:"node"`
	Key            string   `yaml:"key" json:"key"`
	Layer          string   `yaml:"layer" json:"layer"`
	CoreConcepts   []string `yaml:"core_concepts" json:"core_concepts"`
	CommonMistakes []string `yaml:"common_mistakes" json:"common_mistakes"`
	Boundaries     []string `yaml:"boundaries" json:"boundaries"`
	ExerciseIdeas  []string `yaml:"exercise_ideas" json:"exercise_ideas"`
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
	Key   string `yaml:"key" json:"key"`
	Title string `yaml:"title" json:"title"`
}
