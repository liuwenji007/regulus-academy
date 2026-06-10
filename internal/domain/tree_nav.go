package domain

import "github.com/regulus-academy/regulus-academy/internal/storage"

// CompletedKeysFromProgress 收集 status=completed 的节点 key 集合
func CompletedKeysFromProgress(list []storage.UserProgress) map[string]bool {
	completed := make(map[string]bool)
	for _, p := range list {
		if p.Status == "completed" {
			completed[p.NodeKey] = true
		}
	}
	return completed
}

// NextNodeAfter 按知识树 layers 顺序返回当前节点的下一节点
func NextNodeAfter(tree *storage.KnowledgeTree, nodeKey string) (key, layer, title string, ok bool) {
	return nextNodeAfter(tree, nodeKey, nil)
}

// NextUncompletedNodeAfter 按树顺序返回当前节点之后第一个未完成的节点（跳过 completed）
func NextUncompletedNodeAfter(tree *storage.KnowledgeTree, nodeKey string, completed map[string]bool) (key, layer, title string, ok bool) {
	return nextNodeAfter(tree, nodeKey, completed)
}

func nextNodeAfter(tree *storage.KnowledgeTree, nodeKey string, skipCompleted map[string]bool) (key, layer, title string, ok bool) {
	if tree == nil || nodeKey == "" {
		return "", "", "", false
	}
	found := false
	for _, ly := range tree.Layers {
		for _, n := range ly.Nodes {
			if found {
				if skipCompleted != nil && skipCompleted[n.Key] {
					continue
				}
				return n.Key, ly.Key, n.Title, true
			}
			if n.Key == nodeKey {
				found = true
			}
		}
	}
	return "", "", "", false
}
