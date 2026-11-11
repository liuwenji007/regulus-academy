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
	var slug, source, parentSlug, field, fieldSource sql.NullString
	err = s.db.QueryRow(
		`SELECT id, name, slug, source, parent_slug, field, field_source, created_at FROM domains WHERE id = ?`, domainID,
	).Scan(&d.ID, &d.Name, &slug, &source, &parentSlug, &field, &fieldSource, &d.CreatedAt)
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
	if parentSlug.Valid {
		d.ParentSlug = parentSlug.String
	}
	if field.Valid {
		d.Field = field.String
	}
	if fieldSource.Valid {
		d.FieldSource = fieldSource.String
	}
	return &d, nil
}

// UpdateDomainField 写入课程所属领域。source 为 model 或 user。
func (s *Store) UpdateDomainField(userID, domainID, field, source string) error {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("领域不存在")
	}
	_, err = s.db.Exec(
		`UPDATE domains SET field = ?, field_source = ? WHERE id = ? AND COALESCE(user_id, 'default') = ?`,
		field, source, domainID, userID,
	)
	return err
}

// DistinctDomainFields 该用户已经用过的领域名
func (s *Store) DistinctDomainFields(userID string) ([]string, error) {
	userID = normalizeUserID(userID)
	rows, err := s.db.Query(
		`SELECT DISTINCT field FROM domains WHERE COALESCE(user_id, 'default') = ? AND field IS NOT NULL AND field != '' ORDER BY field`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, err
		}
		out = append(out, field)
	}
	return out, rows.Err()
}

// DomainNeedingField 还没有领域、且不是用户手动指定的课程
type DomainNeedingField struct {
	ID   string
	Name string
	Slug string
}

// ListDomainsNeedingField 列出待模型补判的课程
func (s *Store) ListDomainsNeedingField(userID string) ([]DomainNeedingField, error) {
	userID = normalizeUserID(userID)
	rows, err := s.db.Query(`
		SELECT id, name, COALESCE(slug, '')
		FROM domains
		WHERE COALESCE(user_id, 'default') = ?
		  AND (field IS NULL OR field = '')
		  AND COALESCE(field_source, '') != 'user'
		ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DomainNeedingField
	for rows.Next() {
		var item DomainNeedingField
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetDomainDerivationJSON 读取生成课落库的衍生锚点 JSON
func (s *Store) GetDomainDerivationJSON(domainID string) (string, error) {
	var raw sql.NullString
	err := s.db.QueryRow(`SELECT derivation_json FROM domains WHERE id = ?`, domainID).Scan(&raw)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("领域不存在")
	}
	if err != nil {
		return "", err
	}
	if raw.Valid {
		return raw.String, nil
	}
	return "", nil
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
	for _, step := range []struct {
		q    string
		args []any
	}{
		{`DELETE FROM sessions WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM mistakes WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM user_progress WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM user_domain_profiles WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM user_domain_access WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM channel_active_node WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM node_notes WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM aside_messages WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM term_cards WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM knowledge_gaps WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM source_chunks WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM domain_sources WHERE user_id = ? AND domain_id = ?`, []any{userID, domainID}},
		{`DELETE FROM domain_extensions WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM domain_build_jobs WHERE domain_id = ?`, []any{domainID}},
		{`DELETE FROM domains WHERE id = ? AND COALESCE(user_id, 'default') = ?`, []any{domainID, userID}},
	} {
		if _, err := tx.Exec(step.q, step.args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}
