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

const (
	extendNodeMin      = 2
	extendNodeMaxBroad = 8
	extendNodeMaxNarrow = 5
)

// InferScopeFromTree 根据现有节点规模推断领域广度（扩展时 intent 可能未持久化 scope）
func InferScopeFromTree(tree *storage.KnowledgeTree) string {
	n := countTreeNodes(tree)
	switch {
	case n <= 9:
		return ScopeNarrow
	case n >= 14:
		return ScopeBroad
	default:
		return ScopeModerate
	}
}

// extendNodeLimit 按已完成节点数与课程规模计算单次扩展上限（窄主题封顶 5，宽课程最多 8）
func extendNodeLimit(scope string, completedCount, totalNodes int) int {
	max := 3 + completedCount/5
	if totalNodes > 0 {
		if treeScaled := 3 + totalNodes/5; treeScaled > max {
			max = treeScaled
		}
	}
	if max < extendNodeMin {
		max = extendNodeMin
	}
	cap := extendNodeMaxBroad
	if normalizeScope(scope) == ScopeNarrow {
		cap = extendNodeMaxNarrow
	}
	if max > cap {
		max = cap
	}
	return max
}

// ExtendResult 纵深扩展增量结果
type ExtendResult struct {
	AddedNodeKeys []string
	Tree          *storage.KnowledgeTree
	Nodes         map[string]NodeSpec
}

type extendTreeOutput struct {
	Layers  map[string]TreeLayerDef `json:"layers"`
	Nodes   []NodeSpec              `json:"nodes"`
	Modules []TreeModuleDef         `json:"modules,omitempty"`
}

// Extend 在现有知识树上追加进阶节点（仅增量，不改旧节点）
func (b *TreeBuilder) Extend(
	ctx context.Context,
	client llm.Provider,
	intent IntentResult,
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	profile string,
	completedKeys []string,
	goal string,
) (*ExtendResult, error) {
	if !client.Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法扩展知识树")
	}
	if tree == nil {
		return nil, fmt.Errorf("知识树不存在")
	}
	existingKeys := CollectTreeNodeKeys(tree)
	if intent.ScopeBreadth == "" {
		intent.ScopeBreadth = InferScopeFromTree(tree)
	}
	maxNodes := extendNodeLimit(intent.ScopeBreadth, len(completedKeys), len(existingKeys))
	prompt := buildExtendTreePrompt(intent, tree, nodes, profile, completedKeys, goal, maxNodes)

	var out extendTreeOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 知识树扩展设计师。只输出增量 JSON，不得修改或删除已有节点。"},
		{Role: "user", Content: prompt},
	}
	genCtx := observability.WithGeneration(ctx, "domain.extend_tree")
	if err := client.ChatJSON(genCtx, msgs, 0.35, &out); err != nil {
		return nil, fmt.Errorf("纵深扩展生成失败: %w", err)
	}

	deltaTree, deltaNodes, addedKeys, err := validateExtendOutput(existingKeys, out, maxNodes)
	if err != nil {
		return nil, err
	}

	mergedTree, mergedNodes := mergeExtendIntoTree(tree, nodes, deltaTree, deltaNodes)
	issues := collectTreeQualityIssues(mergedTree, mergedNodes, intent)
	if len(issues) > 0 {
		logTreeQualityIssues(issues)
	}
	if TreeCritiqueEnabled() {
		critique, cerr := critiqueTree(ctx, client, mergedTree, mergedNodes, issues, intent)
		if cerr == nil && critique.Severity == "fail" && strings.TrimSpace(critique.Feedback) != "" {
			log.Printf("扩展 critique 警告: %s", critique.Feedback)
		}
	}

	return &ExtendResult{
		AddedNodeKeys: addedKeys,
		Tree:          mergedTree,
		Nodes:         mergedNodes,
	}, nil
}

func buildExtendTreePrompt(
	intent IntentResult,
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	profile string,
	completedKeys []string,
	goal string,
	maxNodes int,
) string {
	var b strings.Builder
	b.WriteString("## 现有知识树摘要\n\n")
	b.WriteString(fmt.Sprintf("主题：%s（%s）\n", intent.DisplayName, intent.Slug))
	if goal != "" {
		b.WriteString("用户扩展目标：")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	if profile != "" {
		b.WriteString("\n【学生画像】\n")
		b.WriteString(profile)
		b.WriteString("\n")
	}
	if len(completedKeys) > 0 {
		b.WriteString("\n已掌握节点：")
		b.WriteString(strings.Join(completedKeys, "、"))
		b.WriteString("\n")
	}

	b.WriteString("\n现有节点 key 列表（禁止重复）：")
	b.WriteString(strings.Join(CollectTreeNodeKeys(tree), "、"))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, `## 任务

用户已完成基础路径，请**仅追加** %d～%d 个节点（不得超过 %d 个），按目标分层放置：
- **advanced（精通）**：原理深挖、源码级、架构与性能等纵深专题
- **intermediate（熟悉）**：生产实战、工程模式、排障与落地（可新建「生产实战」类 module）
- 可选：在末尾追加 1 个 *_capstone 综合实战节点

硬性约束：
- 新节点总数必须 ≤ %d（超出将被系统截断）
- 新 key 不得与现有重复
- 不得修改/删除旧节点
- 每个新节点须有完整 nodes 边界（core_concepts、teaching_beats 等）
- 新节点必须分配到 modules（新建 module 或扩展现有 module 的 nodes 列表）

输出 JSON（可同时含 intermediate 与 advanced）：
{
  "layers": {
    "intermediate": { "label":"熟悉", "nodes":[{"key":"prod_patterns","title":"生产常见模式"}] },
    "advanced": { "label":"精通", "time":"...", "goal":"...", "nodes":[{"key":"new_key","title":"..."}] }
  },
  "nodes": [ { "key":"new_key", "node":"...", "layer":"精通", "core_concepts":["..."], ... } ],
  "modules": [ { "key":"production_practice", "label":"生产实战", "nodes":["prod_patterns"] } ]
}
`, extendNodeMin, maxNodes, maxNodes, maxNodes)
	return b.String()
}

func validateExtendOutput(existingKeys []string, out extendTreeOutput, maxNodes int) (*storage.KnowledgeTree, map[string]NodeSpec, []string, error) {
	existing := map[string]struct{}{}
	for _, k := range existingKeys {
		existing[k] = struct{}{}
	}

	deltaTree := &storage.KnowledgeTree{}
	var addedKeys []string
	order := []string{"entry", "intermediate", "advanced"}

	for _, layerKey := range order {
		layer, ok := out.Layers[layerKey]
		if !ok {
			continue
		}
		var nodes []storage.TreeNode
		for _, n := range layer.Nodes {
			if n.Key == "" {
				return nil, nil, nil, fmt.Errorf("扩展节点缺少 key")
			}
			if _, dup := existing[n.Key]; dup {
				return nil, nil, nil, fmt.Errorf("扩展节点 %s 与现有 key 重复", n.Key)
			}
			existing[n.Key] = struct{}{}
			addedKeys = append(addedKeys, n.Key)
			nodes = append(nodes, storage.TreeNode{Key: n.Key, Title: n.Title})
		}
		if len(nodes) == 0 {
			continue
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
			timeEst = estimateLayerTime(layerKey, len(nodes))
		}
		deltaTree.Layers = append(deltaTree.Layers, storage.TreeLayer{
			Key: layerKey, Label: label, Time: timeEst, Goal: goal, Nodes: nodes,
		})
	}

	if len(addedKeys) == 0 {
		return nil, nil, nil, fmt.Errorf("扩展未产生新节点")
	}
	if maxNodes <= 0 {
		maxNodes = extendNodeMaxNarrow
	}
	if len(addedKeys) > maxNodes {
		log.Printf("扩展产出 %d 个节点，截断至上限 %d", len(addedKeys), maxNodes)
		addedKeys, deltaTree = truncateExtendTree(deltaTree, addedKeys, maxNodes)
		out.Modules = filterExtendModules(out.Modules, addedKeys)
	}

	deltaNodes := make(map[string]NodeSpec, len(out.Nodes))
	for _, spec := range out.Nodes {
		if spec.Key == "" {
			continue
		}
		found := false
		for _, k := range addedKeys {
			if k == spec.Key {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		if len(spec.CoreConcepts) == 0 {
			return nil, nil, nil, fmt.Errorf("节点 %s 缺少 core_concepts", spec.Key)
		}
		deltaNodes[spec.Key] = spec
	}
	for _, k := range addedKeys {
		if _, ok := deltaNodes[k]; !ok {
			return nil, nil, nil, fmt.Errorf("缺少节点边界定义: %s", k)
		}
	}

	if len(out.Modules) > 0 {
		newKeys := map[string]struct{}{}
		for _, k := range addedKeys {
			newKeys[k] = struct{}{}
		}
		if err := validateExtendModules(out.Modules, newKeys); err != nil {
			return nil, nil, nil, err
		}
		deltaTree.Modules = nil
		for _, m := range out.Modules {
			deltaTree.Modules = append(deltaTree.Modules, storage.TreeModule{
				Key: m.Key, Label: m.Label, Goal: m.Goal, Order: m.Order, Nodes: append([]string(nil), m.Nodes...),
			})
		}
	}

	return deltaTree, deltaNodes, addedKeys, nil
}

func truncateExtendTree(delta *storage.KnowledgeTree, addedKeys []string, maxN int) ([]string, *storage.KnowledgeTree) {
	if maxN <= 0 || len(addedKeys) <= maxN {
		return addedKeys, delta
	}
	kept := make(map[string]struct{}, maxN)
	for _, k := range addedKeys[:maxN] {
		kept[k] = struct{}{}
	}
	addedKeys = addedKeys[:maxN]
	for i := range delta.Layers {
		var nodes []storage.TreeNode
		for _, n := range delta.Layers[i].Nodes {
			if _, ok := kept[n.Key]; ok {
				nodes = append(nodes, n)
			}
		}
		delta.Layers[i].Nodes = nodes
	}
	var layers []storage.TreeLayer
	for _, l := range delta.Layers {
		if len(l.Nodes) > 0 {
			layers = append(layers, l)
		}
	}
	delta.Layers = layers
	return addedKeys, delta
}

func filterExtendModules(modules []TreeModuleDef, keptKeys []string) []TreeModuleDef {
	kept := make(map[string]struct{}, len(keptKeys))
	for _, k := range keptKeys {
		kept[k] = struct{}{}
	}
	var out []TreeModuleDef
	for _, m := range modules {
		var nodes []string
		for _, nk := range m.Nodes {
			nk = strings.TrimSpace(nk)
			if _, ok := kept[nk]; ok {
				nodes = append(nodes, nk)
			}
		}
		if len(nodes) == 0 {
			continue
		}
		cp := m
		cp.Nodes = nodes
		out = append(out, cp)
	}
	assigned := make(map[string]struct{})
	for _, m := range out {
		for _, nk := range m.Nodes {
			assigned[nk] = struct{}{}
		}
	}
	var orphans []string
	for _, k := range keptKeys {
		if _, ok := assigned[k]; !ok {
			orphans = append(orphans, k)
		}
	}
	if len(orphans) == 0 {
		return out
	}
	if len(out) > 0 {
		out[len(out)-1].Nodes = append(out[len(out)-1].Nodes, orphans...)
		return out
	}
	return []TreeModuleDef{{
		Key:   "advanced_extra",
		Label: "进阶专题",
		Nodes: orphans,
	}}
}

func validateExtendModules(modules []TreeModuleDef, newKeys map[string]struct{}) error {
	assigned := map[string]string{}
	for i, m := range modules {
		key := strings.TrimSpace(m.Key)
		label := strings.TrimSpace(m.Label)
		if key == "" {
			return fmt.Errorf("扩展模块 %d 缺少 key", i+1)
		}
		if label == "" {
			return fmt.Errorf("扩展模块 %s 缺少 label", key)
		}
		if len(m.Nodes) == 0 {
			return fmt.Errorf("扩展模块 %s 至少包含 1 个节点", key)
		}
		for _, nk := range m.Nodes {
			nk = strings.TrimSpace(nk)
			if nk == "" {
				continue
			}
			if _, ok := newKeys[nk]; !ok {
				return fmt.Errorf("扩展模块 %s 引用了非新增节点 %s", key, nk)
			}
			if prev, dup := assigned[nk]; dup {
				return fmt.Errorf("节点 %s 同时归属模块 %s 与 %s", nk, prev, key)
			}
			assigned[nk] = key
		}
	}
	for nk := range newKeys {
		if _, ok := assigned[nk]; !ok {
			return fmt.Errorf("新增节点 %s 未分配到任何主题模块", nk)
		}
	}
	return nil
}

func mergeExtendIntoTree(
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	deltaTree *storage.KnowledgeTree,
	deltaNodes map[string]NodeSpec,
) (*storage.KnowledgeTree, map[string]NodeSpec) {
	if nodes == nil {
		nodes = map[string]NodeSpec{}
	}
	merged := *tree
	merged.Layers = append([]storage.TreeLayer(nil), tree.Layers...)

	for _, deltaLayer := range deltaTree.Layers {
		layerIdx := -1
		for i := range merged.Layers {
			if merged.Layers[i].Key == deltaLayer.Key {
				layerIdx = i
				break
			}
		}
		if layerIdx < 0 {
			merged.Layers = append(merged.Layers, deltaLayer)
		} else {
			merged.Layers[layerIdx].Nodes = append(merged.Layers[layerIdx].Nodes, deltaLayer.Nodes...)
		}
	}

	for k, spec := range deltaNodes {
		nodes[k] = spec
	}
	MergeNodeRequires(&merged, nodes)

	if len(deltaTree.Modules) > 0 {
		merged.Modules = append(merged.Modules, deltaTree.Modules...)
	} else if len(merged.Modules) > 0 {
		last := len(merged.Modules) - 1
		for _, deltaLayer := range deltaTree.Layers {
			for _, n := range deltaLayer.Nodes {
				merged.Modules[last].Nodes = append(merged.Modules[last].Nodes, n.Key)
			}
		}
	}

	return &merged, nodes
}
