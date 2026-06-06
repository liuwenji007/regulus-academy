package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UpdateDomainTreeInPlace 原地更新知识树并递增 tree_version，写入扩展审计记录
func (s *Store) UpdateDomainTreeInPlace(userID, domainID string, tree *KnowledgeTree, nodesJSON string, addedKeys []string, reason string) (int, error) {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("领域不存在")
	}
	if tree == nil {
		return 0, fmt.Errorf("知识树不能为空")
	}
	if nodesJSON == "" {
		nodesJSON = "{}"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var fromVersion int
	err = tx.QueryRow(`SELECT COALESCE(tree_version, 1) FROM domains WHERE id = ?`, domainID).Scan(&fromVersion)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("领域不存在")
	}
	if err != nil {
		return 0, err
	}
	toVersion := fromVersion + 1

	tree.DomainID = domainID
	treeJSON, err := json.Marshal(tree)
	if err != nil {
		return 0, err
	}
	addedJSON, err := json.Marshal(addedKeys)
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(
		`UPDATE domains SET tree_json = ?, nodes_json = ?, tree_version = ? WHERE id = ? AND COALESCE(user_id, 'default') = ?`,
		string(treeJSON), nodesJSON, toVersion, domainID, userID,
	); err != nil {
		return 0, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec(
		`INSERT INTO domain_extensions (id, domain_id, user_id, from_version, to_version, added_nodes_json, reason, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), domainID, userID, fromVersion, toVersion, string(addedJSON), reason, now,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return toVersion, nil
}

// GetDomainTreeVersion 获取课程树版本号
func (s *Store) GetDomainTreeVersion(domainID string) (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(tree_version, 1) FROM domains WHERE id = ?`, domainID).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("领域不存在")
	}
	return v, err
}
