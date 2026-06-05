package storage

import (
	"database/sql"
	"fmt"
)

// GetDomain 获取课程元信息（需属于该用户）
func (s *Store) GetDomain(userID, domainID string) (*Domain, error) {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("领域不存在")
	}
	var d Domain
	var slug, source sql.NullString
	err = s.db.QueryRow(
		`SELECT id, name, slug, source, created_at FROM domains WHERE id = ?`, domainID,
	).Scan(&d.ID, &d.Name, &slug, &source, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("领域不存在")
	}
	if err != nil {
		return nil, err
	}
	d.UserID = userID
	if slug.Valid {
		d.Slug = slug.String
	}
	if source.Valid {
		d.Source = source.String
	}
	return &d, nil
}

// ClearDomainSlug 清空课程 slug（重建新课程前释放同 slug 唯一约束）。
func (s *Store) ClearDomainSlug(userID, domainID string) error {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("领域不存在")
	}
	_, err = s.db.Exec(`UPDATE domains SET slug = NULL WHERE id = ? AND COALESCE(user_id, 'default') = ?`, domainID, userID)
	return err
}

// DeleteDomain 删除课程及其进度、会话、错题等关联数据
func (s *Store) DeleteDomain(userID, domainID string) error {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("领域不存在")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`DELETE FROM session_messages WHERE session_id IN (SELECT id FROM sessions WHERE domain_id = ?)`,
		domainID,
	); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM sessions WHERE domain_id = ?`,
		`DELETE FROM mistakes WHERE domain_id = ?`,
		`DELETE FROM user_progress WHERE domain_id = ?`,
		`DELETE FROM channel_active_node WHERE user_id = ? AND domain_id = ?`,
		`DELETE FROM domains WHERE id = ? AND COALESCE(user_id, 'default') = ?`,
	} {
		args := []any{domainID}
		if q == `DELETE FROM channel_active_node WHERE user_id = ? AND domain_id = ?` {
			args = []any{userID, domainID}
		}
		if q == `DELETE FROM domains WHERE id = ? AND COALESCE(user_id, 'default') = ?` {
			args = []any{domainID, userID}
		}
		if _, err := tx.Exec(q, args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}
