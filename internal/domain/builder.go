package domain

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
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
	Domain      string                  `json:"domain"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	Modules     []TreeModuleDef         `json:"modules"`
	Layers      map[string]TreeLayerDef `json:"layers"`
	Nodes       []NodeSpec              `json:"nodes"`
}

const (
	ScopeNarrow   = "narrow"
	ScopeModerate = "moderate"
	ScopeBroad    = "broad"
)

var layerDefaults = map[string]struct {
	Label string
	Goal  string
}{
	"entry": {
		Label: "入门",
		Goal:  "快速掌握核心概念，能看懂相关代码与文档，建立该领域的知识框架",
	},
	"intermediate": {
		Label: "熟悉",
		Goal:  "能在真实项目中动手应用，独立应对大多数常见场景",
	},
	"advanced": {
		Label: "精通",
		Goal:  "能排查疑难问题、做架构取舍，覆盖绝大多数复杂场景",
	},
}

// Build 根据意图 LLM 生成知识树与节点边界
func (b *TreeBuilder) Build(ctx context.Context, client llm.Provider, intent IntentResult, userInput string) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	if !client.Configured() {
		return nil, nil, fmt.Errorf("未配置 LLM，无法生成知识树")
	}

	var out buildTreeOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 知识树设计师。根据具体领域为在职开发者设计可执行的三层渐进式学习路径。只输出 JSON。"},
		{Role: "user", Content: buildTreePrompt(intent, userInput)},
	}
	ctx = observability.WithGeneration(ctx, "domain.build_tree")
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

func nodeCountBounds(scope string) (minTotal, maxTotal int) {
	switch normalizeScope(scope) {
	case ScopeNarrow:
		return 5, 9
	case ScopeBroad:
		return 12, 20
	default:
		return 8, 14
	}
}

func normalizeScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ScopeNarrow, "small", "focused":
		return ScopeNarrow
	case ScopeBroad, "large", "wide":
		return ScopeBroad
	default:
		return ScopeModerate
	}
}

func buildTreePrompt(intent IntentResult, userInput string) string {
	core, _ := LoadPrompt("core")
	scope := normalizeScope(intent.ScopeBreadth)
	minTotal, maxTotal := nodeCountBounds(scope)

	var b strings.Builder
	b.WriteString("用户原话：")
	b.WriteString(userInput)
	b.WriteString("\n主题：")
	b.WriteString(intent.DisplayName)
	b.WriteString("\nslug：")
	b.WriteString(intent.Slug)
	b.WriteString("\n领域广度评估：")
	b.WriteString(scope)
	b.WriteString("（")
	b.WriteString(scopeBreadthHint(scope))
	b.WriteString("）\n")
	if intent.Reason != "" {
		b.WriteString("学习意图：")
		b.WriteString(intent.Reason)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if core != "" {
		b.WriteString("学习方式参考：\n")
		b.WriteString(core)
		b.WriteString("\n\n")
	}

	b.WriteString(`## 三层定位（必须体现在各层 goal 中，可结合本主题改写）

- **入门**：快速掌握基础知识，能看懂代码/文档/讨论，建立该领域的知识框架（不是浅尝辄止，而是「看得懂地图」）
- **熟悉**：可以开始动手应用，能独立完成大多数日常/常见场景下的任务
- **精通**：能解决高难度与边界问题，在绝大多数复杂场景下仍能做出正确判断

## 时间与规模

- **time 必须按本主题实际估算**，禁止所有课程都用「~2 小时 / ~8 小时 / ~20 小时」
- 估算依据：节点数量 × 每节约 15 分钟微训练，加上理解消化时间；窄话题可短，宽话题可长
- time 用自然中文，如「约 3 小时」「约 1～2 周（每天 30 分钟）」「约 25～35 小时」
- 本主题建议总节点数：`)
	fmt.Fprintf(&b, "%d～%d 个", minTotal, maxTotal)
	b.WriteString(`，按领域实际拆分，不要凑数

## 主题模块 modules（与 layers 独立）

- **modules** = 知识结构分块（如「基础语法」「方法与接口」「泛型」「并发」），供知识图谱聚类展示
- **layers** = 学习进度深度（入门/熟悉/精通），供课程列表与学习路径使用
- 每个节点 key 必须恰好出现在一个 module 的 nodes 数组中
- module 的 label 用中文主题名，禁止复用「入门」「熟悉」「精通」
- 模块数量：窄主题 2～3 个，中等 3～5 个，宽主题 4～6 个

## JSON 结构

{
  "domain": "中文领域名",
  "slug": "与上文 slug 一致",
  "description": "一句话描述学完能做什么",
  "modules": [
    { "key": "basics", "label": "基础语法", "goal": "可选，一句话说明本模块覆盖什么", "nodes": ["node_key_a", "node_key_b"] },
    { "key": "concurrency", "label": "并发", "goal": "...", "nodes": ["node_key_c"] }
  ],
  "layers": {
    "entry": { "label": "入门", "time": "按主题估算", "goal": "体现「看懂+知识框架」", "nodes": [{"key": "snake_case", "title": "..."}] },
    "intermediate": { "label": "熟悉", "time": "按主题估算", "goal": "体现「能应用+常见场景」", "nodes": [...] },
    "advanced": { "label": "精通", "time": "按主题估算", "goal": "体现「高难度+绝大多数复杂场景」", "nodes": [...] }
  },
  "nodes": [
    {
      "key": "与 layers 中 key 一致",
      "node": "节点中文名",
      "layer": "入门/熟悉/精通 之一",
      "core_concepts": ["..."],
      "common_mistakes": ["..."],
      "boundaries": ["本节点不讲什么"],
      "exercise_ideas": ["可出的练习题方向"],
      "grading_hints": ["可选，批改时对照的评分要点短语"]
    }
  ]
}

## 硬性约束

- 必须包含 modules 数组，且每个节点 key 恰好归属 1 个 module
- 必须包含 entry、intermediate、advanced 三层
- 入门层 2～5 节点，熟悉层 2～6 节点，精通层 1～5 节点（窄主题偏少，宽主题偏多）
- 节点按由浅入深排列；boundaries 标明不越界，避免层与层之间内容重叠
- 当总节点数 ≤ 8：相邻节点 core_concepts 互不重复；boundaries 写明「由哪一节点负责」以免题面重叠
- 每个节点：exercise_ideas 条数 ≥ min(2, core_concepts 条数)，且每条 idea 对应不同 concept
- 每个 layers 中的 key 必须在 nodes 数组中有完整边界定义
- key 用 snake_case 英文`)
	return b.String()
}

func scopeBreadthHint(scope string) string {
	switch scope {
	case ScopeNarrow:
		return "聚焦子话题，如「Go channel」「React Hooks」"
	case ScopeBroad:
		return "宽泛领域，如「Rust 语言」「分布式系统」「Agent 开发」"
	default:
		return "中等范围主题，如「Go 语言」「前端工程化」"
	}
}

func validateBuildOutput(out buildTreeOutput, intent IntentResult) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	order := []string{"entry", "intermediate", "advanced"}
	tree := &storage.KnowledgeTree{DomainName: out.Domain}
	if tree.DomainName == "" {
		tree.DomainName = intent.DisplayName
	}

	minTotal, maxTotal := nodeCountBounds(intent.ScopeBreadth)
	layerMin := map[string]int{"entry": 2, "intermediate": 2, "advanced": 1}
	layerMax := map[string]int{"entry": 5, "intermediate": 6, "advanced": 5}

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
		if len(nodes) < layerMin[layerKey] {
			return nil, nil, fmt.Errorf("层级 %s 至少需要 %d 个节点", layerKey, layerMin[layerKey])
		}
		if len(nodes) > layerMax[layerKey] {
			return nil, nil, fmt.Errorf("层级 %s 最多 %d 个节点", layerKey, layerMax[layerKey])
		}

		def := layerDefaults[layerKey]
		label := strings.TrimSpace(layer.Label)
		if label == "" {
			label = def.Label
		}
		goal := strings.TrimSpace(layer.Goal)
		if goal == "" {
			goal = def.Goal
		}
		timeEst := strings.TrimSpace(layer.Time)
		if timeEst == "" {
			return nil, nil, fmt.Errorf("层级 %s 缺少 time 估算", layerKey)
		}
		if isGenericTime(timeEst) {
			return nil, nil, fmt.Errorf("层级 %s 的 time 过于模板化，请按主题实际估算", layerKey)
		}

		tree.Layers = append(tree.Layers, storage.TreeLayer{
			Key: layerKey, Label: label, Time: timeEst, Goal: goal, Nodes: nodes,
		})
	}

	if len(tree.Layers) != 3 {
		return nil, nil, fmt.Errorf("需要 3 层知识树")
	}
	total := 0
	for _, l := range tree.Layers {
		total += len(l.Nodes)
	}
	if total < minTotal || total > maxTotal {
		return nil, nil, fmt.Errorf("节点总数应在 %d-%d 之间（当前主题广度 %s），得到 %d",
			minTotal, maxTotal, normalizeScope(intent.ScopeBreadth), total)
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

	MergeNodeRequires(tree, nodes)

	modules, err := validateModules(out.Modules, nodeKeys, intent.ScopeBreadth)
	if err != nil {
		return nil, nil, err
	}
	tree.Modules = modules

	warnBuildTreeQuality(nodes, total)

	return tree, nodes, nil
}

// warnBuildTreeQuality 轻量质量提示，不阻断建树。
func warnBuildTreeQuality(nodes map[string]NodeSpec, totalNodes int) {
	seenConcept := map[string]string{}
	for key, spec := range nodes {
		if len(spec.CoreConcepts) >= 2 && len(spec.ExerciseIdeas) < len(spec.CoreConcepts) {
			log.Printf("建树提示: 节点 %s 的 exercise_ideas 少于 core_concepts，建议补全", key)
		}
		for _, c := range spec.CoreConcepts {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if other, ok := seenConcept[c]; ok && other != key {
				log.Printf("建树提示: core_concept %q 在节点 %s 与 %s 重复", c, other, key)
			} else {
				seenConcept[c] = key
			}
		}
	}
	if totalNodes > 0 && totalNodes <= 8 {
		log.Printf("建树提示: 节点数 %d（≤8），请确认相邻节点 boundaries 已区分职责", totalNodes)
	}
}

// isGenericTime 检测是否仍在使用旧版固定时间模板
func isGenericTime(timeEst string) bool {
	t := strings.ReplaceAll(strings.TrimSpace(timeEst), " ", "")
	generic := []string{"~2小时", "~8小时", "~20小时", "约2小时", "约8小时", "约20小时"}
	for _, g := range generic {
		if strings.EqualFold(t, g) {
			return true
		}
	}
	return false
}
