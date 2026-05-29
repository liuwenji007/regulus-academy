package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// TreeBuilder LLM 动态建树
type TreeBuilder struct {
	registry *Registry
}

// NewTreeBuilder 创建建树器
func NewTreeBuilder(r *Registry) *TreeBuilder {
	return &TreeBuilder{registry: r}
}

type buildTreeOutput struct {
	Domain      string                 `json:"domain"`
	Slug        string                 `json:"slug"`
	Description string                 `json:"description"`
	Layers      map[string]TreeLayerDef `json:"layers"`
	Nodes       []NodeSpec             `json:"nodes"`
}

// Build 根据意图 LLM 生成知识树与节点边界
func (b *TreeBuilder) Build(ctx context.Context, client llm.Provider, intent IntentResult, userInput string) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	if !client.Configured() {
		return nil, nil, fmt.Errorf("未配置 LLM，无法生成知识树")
	}

	var out buildTreeOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 知识树设计师。为在职开发者设计三层渐进式学习路径。只输出 JSON。"},
		{Role: "user", Content: buildTreePrompt(intent, userInput)},
	}
	if err := client.ChatJSON(ctx, msgs, 0.4, &out); err != nil {
		return nil, nil, fmt.Errorf("知识树生成失败: %w", err)
	}

	tree, nodes, err := validateBuildOutput(out, intent)
	if err != nil {
		return nil, nil, err
	}
	tree.DomainName = intent.DisplayName
	return tree, nodes, nil
}

func buildTreePrompt(intent IntentResult, userInput string) string {
	protocol, _ := LoadProtocol()
	var b strings.Builder
	b.WriteString("用户原话：")
	b.WriteString(userInput)
	b.WriteString("\n主题：")
	b.WriteString(intent.DisplayName)
	b.WriteString("\nslug：")
	b.WriteString(intent.Slug)
	b.WriteString("\n\n")
	if protocol != "" {
		b.WriteString("学习方式参考：\n")
		b.WriteString(protocol)
		b.WriteString("\n\n")
	}
	b.WriteString(`请输出 JSON，结构如下：
{
  "domain": "中文领域名",
  "slug": "与上文 slug 一致",
  "description": "一句话描述",
  "layers": {
    "entry": { "label": "入门", "time": "~2 小时", "goal": "...", "nodes": [{"key": "snake_case", "title": "..."}] },
    "intermediate": { "label": "熟悉", "time": "~8 小时", "goal": "...", "nodes": [...] },
    "advanced": { "label": "精通", "time": "~20 小时", "goal": "...", "nodes": [...] }
  },
  "nodes": [
    {
      "key": "与 layers 中 key 一致",
      "node": "节点中文名",
      "layer": "入门/熟悉/精通 之一",
      "core_concepts": ["..."],
      "common_mistakes": ["..."],
      "boundaries": ["本节点不讲什么"],
      "exercise_ideas": ["可出的练习题方向"]
    }
  ]
}

约束：
- 必须包含 entry、intermediate、advanced 三层
- 每层 2-4 个节点，总共 6-10 个节点
- 每个 layers 中的 key 必须在 nodes 数组中有完整边界定义
- key 用 snake_case 英文
- 节点按由浅入深排列，boundaries 标明不越界`)
	return b.String()
}

func validateBuildOutput(out buildTreeOutput, intent IntentResult) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	order := []string{"entry", "intermediate", "advanced"}
	tree := &storage.KnowledgeTree{DomainName: out.Domain}
	if tree.DomainName == "" {
		tree.DomainName = intent.DisplayName
	}

	nodeKeys := map[string]struct{}{}
	for _, layerKey := range order {
		layer, ok := out.Layers[layerKey]
		if !ok {
			return nil, nil, fmt.Errorf("缺少层级 %s", layerKey)
		}
		nodes := make([]storage.TreeNode, len(layer.Nodes))
		for i, n := range layer.Nodes {
			if n.Key == "" {
				return nil, nil, fmt.Errorf("层级 %s 存在空 key", layerKey)
			}
			nodeKeys[n.Key] = struct{}{}
			nodes[i] = storage.TreeNode{Key: n.Key, Title: n.Title}
		}
		if len(nodes) < 2 {
			return nil, nil, fmt.Errorf("层级 %s 至少需要 2 个节点", layerKey)
		}
		tree.Layers = append(tree.Layers, storage.TreeLayer{
			Key: layerKey, Label: layer.Label, Time: layer.Time, Goal: layer.Goal, Nodes: nodes,
		})
	}

	if len(tree.Layers) != 3 {
		return nil, nil, fmt.Errorf("需要 3 层知识树")
	}
	total := 0
	for _, l := range tree.Layers {
		total += len(l.Nodes)
	}
	if total < 6 || total > 12 {
		return nil, nil, fmt.Errorf("节点总数应在 6-12 之间，得到 %d", total)
	}

	nodes := make(map[string]NodeSpec, len(out.Nodes))
	for _, spec := range out.Nodes {
		if spec.Key == "" {
			continue
		}
		if _, ok := nodeKeys[spec.Key]; !ok {
			return nil, nil, fmt.Errorf("节点 %s 不在知识树中", spec.Key)
		}
		if len(spec.CoreConcepts) == 0 {
			return nil, nil, fmt.Errorf("节点 %s 缺少 core_concepts", spec.Key)
		}
		if len(spec.ExerciseIdeas) == 0 {
			return nil, nil, fmt.Errorf("节点 %s 缺少 exercise_ideas", spec.Key)
		}
		nodes[spec.Key] = spec
	}
	for key := range nodeKeys {
		if _, ok := nodes[key]; !ok {
			return nil, nil, fmt.Errorf("缺少节点边界定义: %s", key)
		}
	}
	return tree, nodes, nil
}
