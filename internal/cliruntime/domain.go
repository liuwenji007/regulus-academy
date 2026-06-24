package cliruntime

import (
	"encoding/json"
	"fmt"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func marshalNodesJSON(nodes map[string]domain.NodeSpec) (string, error) {
	if len(nodes) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(nodes)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// EnsureDomainFromSlug 确保 domains/<slug> 已注册到本地 SQLite。
func (rt *Runtime) EnsureDomainFromSlug(slug string) (*storage.Domain, *storage.KnowledgeTree, error) {
	slug = domain.Slugify(slug)
	if slug == "" {
		return nil, nil, fmt.Errorf("无效 slug")
	}
	if dom, tree, err := rt.store.GetDomainBySlug(rt.UserID(), slug); err == nil {
		return dom, tree, nil
	}
	tree, nodes, err := rt.registry.LoadTreeAndNodes(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("domains/%s 不存在: %w", slug, err)
	}
	nodesJSON, err := marshalNodesJSON(nodes)
	if err != nil {
		return nil, nil, err
	}
	name := tree.DomainName
	if name == "" {
		name = slug
	}
	parentSlug := ""
	return rt.store.CreateDomainFromTree(
		rt.UserID(), name, slug, parentSlug, tree, nodesJSON, storage.DomainSourceSkillPack, false, "",
	)
}

// ResolveSlug 将用户输入解析为 slug（匹配内置域或 slugify）。
func (rt *Runtime) ResolveSlug(input string) (string, error) {
	if slug, ok := rt.registry.MatchDomain(input); ok {
		return slug, nil
	}
	if meta, ok := rt.registry.FindDomainBySlug(domain.Slugify(input)); ok {
		return meta.Slug, nil
	}
	s := domain.Slugify(input)
	if s == "" {
		return "", fmt.Errorf("无法解析主题: %s", input)
	}
	return s, nil
}

// PickStartNode 选择首个未完成节点，若全部完成则返回第一个节点。
func PickStartNode(tree *storage.KnowledgeTree, progress []storage.UserProgress) (nodeKey, layer string, err error) {
	if tree == nil || len(tree.Layers) == 0 {
		return "", "", fmt.Errorf("知识树为空")
	}
	completed := domain.CompletedKeysFromProgress(progress)
	for _, ly := range tree.Layers {
		for _, n := range ly.Nodes {
			if !completed[n.Key] {
				return n.Key, ly.Key, nil
			}
		}
	}
	ly := tree.Layers[0]
	if len(ly.Nodes) == 0 {
		return "", "", fmt.Errorf("知识树无节点")
	}
	return ly.Nodes[0].Key, ly.Key, nil
}
