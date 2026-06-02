package domain

import "github.com/regulus-academy/regulus-academy/internal/storage"

// NextNodeAfter 按知识树 layers 顺序返回当前节点的下一节点
func NextNodeAfter(tree *storage.KnowledgeTree, nodeKey string) (key, layer, title string, ok bool) {
	if tree == nil || nodeKey == "" {
		return "", "", "", false
	}
	found := false
	for _, ly := range tree.Layers {
		for _, n := range ly.Nodes {
			if found {
				return n.Key, ly.Key, n.Title, true
			}
			if n.Key == nodeKey {
				found = true
			}
		}
	}
	return "", "", "", false
}
