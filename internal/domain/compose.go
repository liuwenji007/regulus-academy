package domain

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// RootDisplayName 主题根的中文名
func RootDisplayName(rootSlug string) string {
	switch strings.ToLower(strings.TrimSpace(rootSlug)) {
	case "go":
		return "Go 语言"
	case "rust":
		return "Rust"
	default:
		if rootSlug == "" {
			return ""
		}
		return rootSlug
	}
}

// SkillPackNodeKeys 返回 Skill 包全部节点 key 及显示名
func (r *Registry) SkillPackNodeKeys(slug string) ([]string, string, error) {
	tree, err := r.LoadTree(slug)
	if err != nil {
		return nil, "", err
	}
	meta, _ := r.FindDomainBySlug(slug)
	var keys []string
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			keys = append(keys, n.Key)
		}
	}
	label := meta.Name
	if label == "" {
		label = tree.DomainName
	}
	return keys, label, nil
}

// NormalizeToRootTree 将子话题意图归并到主题根（如 go-concurrency → go）
func (r *Registry) NormalizeToRootTree(intent IntentResult) IntentResult {
	parent := strings.TrimSpace(r.ParentSlug(intent.Slug))
	if intent.Source == SourceSkillPack && parent != "" {
		keys, label, err := r.SkillPackNodeKeys(intent.Slug)
		if err == nil {
			intent.FocusSlug = intent.Slug
			intent.FocusNodeKeys = keys
			intent.FocusLabel = label
		}
		intent.RootSlug = parent
		intent.Slug = parent
		intent.DisplayName = RootDisplayName(parent)
		intent.ScopeBreadth = ScopeBroad
		intent.Source = SourceGenerated
		if intent.Reason == "" {
			intent.Reason = fmt.Sprintf("在「%s」知识树中聚焦「%s」", intent.DisplayName, label)
		}
		return intent
	}

	root := TopicRoot(intent.Slug)
	if root == "" {
		root = intent.Slug
	}
	intent.RootSlug = root
	if root != intent.Slug {
		intent.Slug = root
		if name := RootDisplayName(root); name != "" && name != root {
			intent.DisplayName = name
		}
	}
	return intent
}

// MergeSkillPackIntoTree 把子话题 Skill 包节点并入根知识树，返回聚焦节点 keys
func MergeSkillPackIntoTree(
	root *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	pack *storage.KnowledgeTree,
	packNodes map[string]NodeSpec,
) []string {
	if root == nil || pack == nil {
		return nil
	}
	if nodes == nil {
		nodes = map[string]NodeSpec{}
	}

	existing := map[string]struct{}{}
	for _, layer := range root.Layers {
		for _, n := range layer.Nodes {
			existing[n.Key] = struct{}{}
		}
	}

	var focusKeys []string
	for _, packLayer := range pack.Layers {
		layerIdx := -1
		for i := range root.Layers {
			if root.Layers[i].Key == packLayer.Key {
				layerIdx = i
				break
			}
		}
		if layerIdx < 0 {
			root.Layers = append(root.Layers, packLayer)
			for _, n := range packLayer.Nodes {
				focusKeys = append(focusKeys, n.Key)
				existing[n.Key] = struct{}{}
				if spec, ok := packNodes[n.Key]; ok {
					nodes[n.Key] = spec
				}
			}
			continue
		}
		for _, pn := range packLayer.Nodes {
			focusKeys = append(focusKeys, pn.Key)
			if _, ok := existing[pn.Key]; !ok {
				root.Layers[layerIdx].Nodes = append(root.Layers[layerIdx].Nodes, pn)
				existing[pn.Key] = struct{}{}
			}
			if spec, ok := packNodes[pn.Key]; ok {
				nodes[pn.Key] = spec
			}
		}
	}
	return focusKeys
}

// CollectTreeNodeKeys 收集树上全部节点 key
func CollectTreeNodeKeys(tree *storage.KnowledgeTree) []string {
	if tree == nil {
		return nil
	}
	var keys []string
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			keys = append(keys, n.Key)
		}
	}
	return keys
}
