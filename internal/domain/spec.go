package domain

// NodeSpec 节点边界定义（来自 nodes/*.yaml）
type NodeSpec struct {
	Node           string   `yaml:"node"`
	Key            string   `yaml:"key"`
	Layer          string   `yaml:"layer"`
	CoreConcepts   []string `yaml:"core_concepts"`
	CommonMistakes []string `yaml:"common_mistakes"`
	Boundaries     []string `yaml:"boundaries"`
	ExerciseIdeas  []string `yaml:"exercise_ideas"`
}

// TreeFile tree.yaml 结构
type TreeFile struct {
	Domain      string                 `yaml:"domain"`
	Slug        string                 `yaml:"slug"`
	Description string                 `yaml:"description"`
	Layers      map[string]TreeLayerDef `yaml:"layers"`
}

// TreeLayerDef 层级定义
type TreeLayerDef struct {
	Label string          `yaml:"label"`
	Time  string          `yaml:"time"`
	Goal  string          `yaml:"goal"`
	Nodes []TreeNodeDef   `yaml:"nodes"`
}

// TreeNodeDef 节点引用
type TreeNodeDef struct {
	Key   string `yaml:"key"`
	Title string `yaml:"title"`
}
