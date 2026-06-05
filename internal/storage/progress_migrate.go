package storage

import (
	"fmt"
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

// MigrateProgressByNodeKey 将旧域 completed 进度迁移到新域（仅 node_key 精确匹配）。
func (s *Store) MigrateProgressByNodeKey(
	userID, fromDomainID, toDomainID string,
	validNewKeys map[string]struct{},
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

	var migrated int
	for _, p := range completed {
		if _, ok := validNewKeys[p.NodeKey]; !ok {
			continue
		}
		_, err := tx.Exec(`
			INSERT INTO user_progress (user_id, domain_id, node_key, layer, status, mastery, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, domain_id, node_key) DO UPDATE SET
				layer=excluded.layer, status=excluded.status, mastery=excluded.mastery, updated_at=excluded.updated_at`,
			userID, toDomainID, p.NodeKey, p.Layer, p.Status, p.Mastery, p.UpdatedAt.UTC(),
		)
		if err != nil {
			return ProgressMigrateResult{}, err
		}
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
