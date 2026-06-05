package storage

import (
	"fmt"
)

// SessionMigrateResult 会话迁移结果。
type SessionMigrateResult struct {
	Migrated int `json:"migrated"`
	Skipped  int `json:"skipped"`
}

// MigrateSessionsByNodeKey 将旧域教学会话（含消息）迁到新域，按 node_key / 标题映射更新 domain 与节点。
// 须在 DeleteDomain(旧域) 之前调用，以便 session_messages 随会话保留。
func (s *Store) MigrateSessionsByNodeKey(
	userID, fromDomainID, toDomainID, newSlug string,
	validNewKeys map[string]struct{},
	oldTree, newTree *KnowledgeTree,
) (SessionMigrateResult, error) {
	userID = normalizeUserID(userID)
	if fromDomainID == "" || toDomainID == "" {
		return SessionMigrateResult{}, fmt.Errorf("领域 ID 不能为空")
	}
	if fromDomainID == toDomainID {
		return SessionMigrateResult{}, fmt.Errorf("源域与目标域不能相同")
	}

	rows, err := s.db.Query(
		`SELECT id, node_key FROM sessions WHERE user_id = ? AND domain_id = ?`,
		userID, fromDomainID,
	)
	if err != nil {
		return SessionMigrateResult{}, err
	}
	defer rows.Close()

	type row struct {
		id, nodeKey string
	}
	var sessions []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.nodeKey); err != nil {
			return SessionMigrateResult{}, err
		}
		sessions = append(sessions, r)
	}
	if err := rows.Err(); err != nil {
		return SessionMigrateResult{}, err
	}

	oldTitleByKey := treeKeyTitleMap(oldTree)
	newKeyByTitle := treeTitleKeyMap(newTree, validNewKeys)

	tx, err := s.db.Begin()
	if err != nil {
		return SessionMigrateResult{}, err
	}
	defer tx.Rollback()

	var migrated int
	for _, sess := range sessions {
		targetKey, ok := resolveMigratedNodeKey(sess.nodeKey, validNewKeys, oldTitleByKey, newKeyByTitle)
		if !ok {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE sessions SET domain_id = ?, domain_slug = ?, node_key = ? WHERE id = ?`,
			toDomainID, newSlug, targetKey, sess.id,
		); err != nil {
			return SessionMigrateResult{}, err
		}
		migrated++
	}

	var activeDomain, activeKey string
	if scanErr := tx.QueryRow(
		`SELECT domain_id, node_key FROM channel_active_node WHERE user_id = ?`,
		userID,
	).Scan(&activeDomain, &activeKey); scanErr == nil && activeDomain == fromDomainID {
		if key, ok := resolveMigratedNodeKey(activeKey, validNewKeys, oldTitleByKey, newKeyByTitle); ok {
			if _, err := tx.Exec(
				`UPDATE channel_active_node SET domain_id = ?, node_key = ?, updated_at = datetime('now') WHERE user_id = ?`,
				toDomainID, key, userID,
			); err != nil {
				return SessionMigrateResult{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return SessionMigrateResult{}, err
	}

	return SessionMigrateResult{
		Migrated: migrated,
		Skipped:  len(sessions) - migrated,
	}, nil
}
