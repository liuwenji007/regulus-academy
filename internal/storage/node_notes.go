package storage

import (
	"database/sql"
	"time"
)

// UpsertNodeNote 写入或更新节点学习笔记
func (s *Store) UpsertNodeNote(userID, domainID, nodeKey, contentMD string) error {
	_, err := s.db.Exec(`
		INSERT INTO node_notes (user_id, domain_id, node_key, content_md, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, domain_id, node_key) DO UPDATE SET
			content_md = excluded.content_md, updated_at = excluded.updated_at`,
		userID, domainID, nodeKey, contentMD, time.Now().UTC(),
	)
	return err
}

// GetNodeNote 读取单条节点笔记；未找到时返回空字符串和 nil error
func (s *Store) GetNodeNote(userID, domainID, nodeKey string) (string, error) {
	var content string
	err := s.db.QueryRow(
		`SELECT content_md FROM node_notes WHERE user_id = ? AND domain_id = ? AND node_key = ?`,
		userID, domainID, nodeKey,
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return content, err
}

// ListNodeNotes 读取 domain 下所有节点笔记，返回 node_key → content_md 映射
func (s *Store) ListNodeNotes(userID, domainID string) (map[string]string, error) {
	rows, err := s.db.Query(
		`SELECT node_key, content_md FROM node_notes WHERE user_id = ? AND domain_id = ?`,
		userID, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// ListMistakesByNode 按 node_key 聚合错题概念，返回 node_key → []concept
func (s *Store) ListMistakesByNode(userID, domainID string) (map[string][]string, error) {
	rows, err := s.db.Query(
		`SELECT node_key, concept FROM mistakes WHERE user_id = ? AND domain_id = ? ORDER BY node_key, last_wrong DESC`,
		userID, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]string)
	for rows.Next() {
		var nodeKey, concept string
		if err := rows.Scan(&nodeKey, &concept); err != nil {
			return nil, err
		}
		result[nodeKey] = append(result[nodeKey], concept)
	}
	return result, rows.Err()
}
