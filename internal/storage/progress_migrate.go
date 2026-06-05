package storage

import (
	"fmt"
	"strings"
)

// ProgressMigrateResult 按 node_key 迁移 completed 进度的结果。
type ProgressMigrateResult struct {
	Migrated int `json:"migrated"`
	Skipped  int `json:"skipped"`
}

// CountCompletedProgress 统计某课程下 completed 节点数。
func (s *Store) CountCompletedProgress(userID, domainID string) (int, error) {
	return s.countCompletedNodes(normalizeUserID(userID), domainID)
}

// MigrateProgressByNodeKey 将旧域 completed 进度迁移到新域。
// 优先 node_key 精确匹配；若提供 oldTree/newTree 且 key 未命中，再按节点标题匹配。
func (s *Store) MigrateProgressByNodeKey(
	userID, fromDomainID, toDomainID string,
	validNewKeys map[string]struct{},
	oldTree, newTree *KnowledgeTree,
) (ProgressMigrateResult, error) {
	userID = normalizeUserID(userID)
	if fromDomainID == "" || toDomainID == "" {
		return ProgressMigrateResult{}, fmt.Errorf("领域 ID 不能为空")
	}
	if fromDomainID == toDomainID {
		return ProgressMigrateResult{}, fmt.Errorf("源域与目标域不能相同")
	}

	list, err := s.ListProgress(userID, fromDomainID)
	if err != nil {
		return ProgressMigrateResult{}, err
	}

	var completed []UserProgress
	for _, p := range list {
		if p.Status == "completed" {
			completed = append(completed, p)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ProgressMigrateResult{}, err
	}
	defer tx.Rollback()

	oldTitleByKey := treeKeyTitleMap(oldTree)
	newKeyByTitle := treeTitleKeyMap(newTree, validNewKeys)

	var migrated int
	usedNewKeys := make(map[string]struct{}, len(completed))
	for _, p := range completed {
		targetKey, ok := resolveMigratedNodeKey(p.NodeKey, validNewKeys, oldTitleByKey, newKeyByTitle)
		if !ok {
			continue
		}
		if _, dup := usedNewKeys[targetKey]; dup {
			continue
		}
		_, err := tx.Exec(`
			INSERT INTO user_progress (user_id, domain_id, node_key, layer, status, mastery, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, domain_id, node_key) DO UPDATE SET
				layer=excluded.layer, status=excluded.status, mastery=excluded.mastery, updated_at=excluded.updated_at`,
			userID, toDomainID, targetKey, p.Layer, p.Status, p.Mastery, p.UpdatedAt.UTC(),
		)
		if err != nil {
			return ProgressMigrateResult{}, err
		}
		usedNewKeys[targetKey] = struct{}{}
		migrated++
	}

	if err := tx.Commit(); err != nil {
		return ProgressMigrateResult{}, err
	}

	return ProgressMigrateResult{
		Migrated: migrated,
		Skipped:  len(completed) - migrated,
	}, nil
}

func normalizeProgressTitle(title string) string {
	return strings.TrimSpace(title)
}

// resolveMigratedNodeKey 将旧树 node_key 映射到新树有效 key（精确匹配优先，否则按标题回退）。
func resolveMigratedNodeKey(
	oldKey string,
	validNewKeys map[string]struct{},
	oldTitleByKey, newKeyByTitle map[string]string,
) (targetKey string, ok bool) {
	targetKey = strings.TrimSpace(oldKey)
	if _, hit := validNewKeys[targetKey]; !hit {
		if title := oldTitleByKey[targetKey]; title != "" {
			if nk := newKeyByTitle[normalizeProgressTitle(title)]; nk != "" {
				targetKey = nk
			}
		}
	}
	if _, hit := validNewKeys[targetKey]; !hit {
		return "", false
	}
	return targetKey, true
}

func treeKeyTitleMap(tree *KnowledgeTree) map[string]string {
	if tree == nil {
		return nil
	}
	out := make(map[string]string)
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			key := strings.TrimSpace(n.Key)
			title := normalizeProgressTitle(n.Title)
			if key == "" || title == "" {
				continue
			}
			out[key] = title
		}
	}
	return out
}

func treeTitleKeyMap(tree *KnowledgeTree, validKeys map[string]struct{}) map[string]string {
	if tree == nil || len(validKeys) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			key := strings.TrimSpace(n.Key)
			title := normalizeProgressTitle(n.Title)
			if key == "" || title == "" {
				continue
			}
			if _, ok := validKeys[key]; !ok {
				continue
			}
			norm := normalizeProgressTitle(title)
			if _, exists := out[norm]; !exists {
				out[norm] = key
			}
		}
	}
	return out
}
