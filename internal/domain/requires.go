package domain

import (
	"encoding/json"
	"fmt"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// MergeNodeRequires 将 nodes 中的 requires 合并进知识树节点（API 响应用）
func MergeNodeRequires(tree *storage.KnowledgeTree, nodes map[string]NodeSpec) {
	if tree == nil || len(nodes) == 0 {
		return
	}
	for li := range tree.Layers {
		for ni := range tree.Layers[li].Nodes {
			key := tree.Layers[li].Nodes[ni].Key
			spec, ok := nodes[key]
			if !ok || len(spec.Requires) == 0 {
				continue
			}
			tree.Layers[li].Nodes[ni].Requires = append([]string(nil), spec.Requires...)
		}
	}
}

// UnmetRequireKeys 返回尚未 completed 的前置 key 列表
func UnmetRequireKeys(requires []string, progress []storage.UserProgress) []string {
	if len(requires) == 0 {
		return nil
	}
	done := map[string]bool{}
	for _, p := range progress {
		if p.Status == "completed" {
			done[p.NodeKey] = true
		}
	}
	var unmet []string
	for _, r := range requires {
		if !done[r] {
			unmet = append(unmet, r)
		}
	}
	return unmet
}

// UnmetRequires 返回尚未 completed 的前置节点 key
func UnmetRequires(tree *storage.KnowledgeTree, nodeKey string, progress []storage.UserProgress) []string {
	return UnmetRequireKeys(requiresForNode(tree, nodeKey), progress)
}

// UnmetRequireTitles 返回未完成前置节点的展示标题
func UnmetRequireTitles(tree *storage.KnowledgeTree, nodeKey string, progress []storage.UserProgress) []string {
	unmet := UnmetRequires(tree, nodeKey, progress)
	if len(unmet) == 0 {
		return nil
	}
	out := make([]string, 0, len(unmet))
	for _, key := range unmet {
		out = append(out, NodeTitle(tree, key))
	}
	return out
}

func requiresForNode(tree *storage.KnowledgeTree, nodeKey string) []string {
	if tree == nil {
		return nil
	}
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			if n.Key == nodeKey {
				return n.Requires
			}
		}
	}
	return nil
}

// LoadDomainNodes 加载领域全部节点边界（Skill 包优先读文件，否则 nodes_json）
func (r *Registry) LoadDomainNodes(store *storage.Store, domainID, slug string) (map[string]NodeSpec, error) {
	if slug != "" {
		if tree, nodes, err := r.LoadTreeAndNodes(slug); err == nil && tree != nil {
			return nodes, nil
		}
	}
	raw, err := store.GetDomainNodesJSON(domainID)
	if err != nil {
		return nil, err
	}
	if raw == "" || raw == "{}" {
		return map[string]NodeSpec{}, nil
	}
	var nodes map[string]NodeSpec
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, fmt.Errorf("解析节点边界失败: %w", err)
	}
	return nodes, nil
}
