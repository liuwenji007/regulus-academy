package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// PersonalSelection 裁剪结果
type PersonalSelection struct {
	// Selected 选中的节点 key 列表（公共树子集）
	Selected []string `json:"selected"`
	// Order 学习顺序（Selected 的排列）
	Order []string `json:"order"`
	// Emphasis key → 一句话说明为什么重点
	Emphasis map[string]string `json:"emphasis,omitempty"`
	// Reason 给用户看的裁剪理由
	Reason string `json:"reason"`
	// RefSlug 引用的公共树 slug
	RefSlug string `json:"refSlug"`
	// RefVersion 引用时的公共树版本号
	RefVersion int `json:"refVersion"`
}

type personalizeLLMOutput struct {
	Selected []string          `json:"selected"`
	Order    []string          `json:"order"`
	Emphasis map[string]string `json:"emphasis,omitempty"`
	Reason   string            `json:"reason"`
}

// Personalize 根据公共知识树和用户背景/目标，让模型挑选、排序、标重点。
// 模型只输出节点 key，不能新增不存在的节点（校验 selected ⊆ 公共树）。
func Personalize(
	ctx context.Context,
	client llm.Provider,
	publicTree *storage.KnowledgeTree,
	treeMeta DomainMeta,
	treeVersion int,
	profile string,
	goal string,
) (*PersonalSelection, error) {
	if !client.Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法裁剪知识树")
	}

	// 构建节点清单：只传 key + title + layer，不传完整边界，省 token
	var briefs []nodeBriefItem
	validKeys := map[string]struct{}{}
	for _, layer := range publicTree.Layers {
		for _, n := range layer.Nodes {
			briefs = append(briefs, nodeBriefItem{Key: n.Key, Title: n.Title, Layer: layer.Label})
			validKeys[n.Key] = struct{}{}
		}
	}

	prompt := buildPersonalizePrompt(treeMeta, briefs, profile, goal)
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 的学习路径顾问。根据用户背景和目标，从公共知识树中挑选最适合的节点并排序。只输出 JSON，不能编造节点。"},
		{Role: "user", Content: prompt},
	}

	var out personalizeLLMOutput
	if err := client.ChatJSON(ctx, msgs, 0.3, &out); err != nil {
		return nil, fmt.Errorf("个性化裁剪失败: %w", err)
	}

	// 校验：selected 必须是公共树子集
	var cleaned []string
	for _, key := range out.Selected {
		if _, ok := validKeys[key]; ok {
			cleaned = append(cleaned, key)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("裁剪结果为空，未匹配任何有效节点")
	}

	// order 也校验，去掉不在 cleaned 中的 key
	cleanedSet := map[string]struct{}{}
	for _, k := range cleaned {
		cleanedSet[k] = struct{}{}
	}
	var orderedKeys []string
	seen := map[string]struct{}{}
	for _, key := range out.Order {
		if _, ok := cleanedSet[key]; ok {
			if _, dup := seen[key]; !dup {
				orderedKeys = append(orderedKeys, key)
				seen[key] = struct{}{}
			}
		}
	}
	// 补齐 order 里没有的 selected 节点（按原公共树层序追加）
	for _, b := range briefs {
		if _, ok := cleanedSet[b.Key]; ok {
			if _, already := seen[b.Key]; !already {
				orderedKeys = append(orderedKeys, b.Key)
				seen[b.Key] = struct{}{}
			}
		}
	}

	return &PersonalSelection{
		Selected:   cleaned,
		Order:      orderedKeys,
		Emphasis:   out.Emphasis,
		Reason:     out.Reason,
		RefSlug:    treeMeta.Slug,
		RefVersion: treeVersion,
	}, nil
}

// ApplySelection 将裁剪结果叠加到公共知识树，返回个性化的 KnowledgeTree
func ApplySelection(publicTree *storage.KnowledgeTree, sel *PersonalSelection) *storage.KnowledgeTree {
	// 建立 key → order 索引
	orderIdx := map[string]int{}
	for i, k := range sel.Order {
		orderIdx[k] = i
	}
	selectedSet := map[string]struct{}{}
	for _, k := range sel.Selected {
		selectedSet[k] = struct{}{}
	}

	personal := &storage.KnowledgeTree{
		DomainID:   publicTree.DomainID,
		DomainName: publicTree.DomainName,
	}

	// 收集各层选中节点，按全局 order 排序
	type layeredNode struct {
		layerIdx int
		orderIdx int
		node     storage.TreeNode
		layer    storage.TreeLayer
	}
	var allNodes []layeredNode
	for li, layer := range publicTree.Layers {
		for _, n := range layer.Nodes {
			if _, ok := selectedSet[n.Key]; !ok {
				continue
			}
			oi, hasOrder := orderIdx[n.Key]
			if !hasOrder {
				oi = 9999
			}
			allNodes = append(allNodes, layeredNode{
				layerIdx: li, orderIdx: oi,
				node: n, layer: layer,
			})
		}
	}

	// 按原层顺序重建 layers（保留各层 label/time/goal），节点按 order 排序
	layerMap := map[string]*storage.TreeLayer{}
	layerOrder := []string{}
	for _, ln := range allNodes {
		lk := ln.layer.Key
		if _, exists := layerMap[lk]; !exists {
			layerOrder = append(layerOrder, lk)
			copied := ln.layer
			copied.Nodes = nil
			layerMap[lk] = &copied
		}
	}

	// 按 order 插入节点到各层
	type withOrder struct {
		oi   int
		node storage.TreeNode
		lk   string
	}
	var sorted []withOrder
	for _, ln := range allNodes {
		sorted = append(sorted, withOrder{oi: ln.orderIdx, node: ln.node, lk: ln.layer.Key})
	}
	// 稳定排序
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].oi < sorted[j-1].oi; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	for _, s := range sorted {
		if layer, ok := layerMap[s.lk]; ok {
			layer.Nodes = append(layer.Nodes, s.node)
		}
	}

	for _, lk := range layerOrder {
		if layer, ok := layerMap[lk]; ok && len(layer.Nodes) > 0 {
			personal.Layers = append(personal.Layers, *layer)
		}
	}

	return personal
}

// SelectionToJSON 序列化裁剪结果为存储用 JSON
func SelectionToJSON(sel *PersonalSelection) (string, error) {
	b, err := json.Marshal(sel)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SelectionFromJSON 从存储 JSON 反序列化裁剪结果
func SelectionFromJSON(raw string) (*PersonalSelection, error) {
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var sel PersonalSelection
	if err := json.Unmarshal([]byte(raw), &sel); err != nil {
		return nil, err
	}
	return &sel, nil
}

type nodeBriefItem struct {
	Key   string
	Title string
	Layer string
}

func buildPersonalizePrompt(meta DomainMeta, briefs []nodeBriefItem, profile, goal string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "领域：%s（%s）\n", meta.Name, meta.Description)
	b.WriteString("全部节点：\n")
	for _, n := range briefs {
		fmt.Fprintf(&b, "  - key=%q  标题=%q  层级=%s\n", n.Key, n.Title, n.Layer)
	}
	if profile != "" {
		fmt.Fprintf(&b, "\n用户背景：%s\n", profile)
	}
	if goal != "" {
		fmt.Fprintf(&b, "学习目标：%s\n", goal)
	}
	b.WriteString(`
请根据用户背景和目标，从上述节点中挑选最适合的子集，并给出推荐学习顺序。

输出 JSON：
{
  "selected": ["key1", "key2", ...],    // 选中的节点 key，必须全部来自上述列表
  "order": ["key1", "key2", ...],       // 推荐学习顺序（selected 的排列）
  "emphasis": {"key1": "为什么重点"},    // 可选，标注需要重点练习的节点
  "reason": "一句话说明为什么这样裁剪"
}

规则：
- selected 只能包含上述 key，不能编造新 key
- 若用户是初学者，保留基础节点；若有基础，可跳过已知内容
- 至少保留 3 个节点，至多保留全部节点
- order 中的顺序决定学习路径，可以打乱层级顺序
`)
	return b.String()
}
