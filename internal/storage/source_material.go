package storage

import (
	"database/sql"
	"fmt"
	"strings"
)

// DomainSourceMaterial 导入建课时抽出的原始正文
type DomainSourceMaterial struct {
	DomainID  string `json:"domainId"`
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	Text      string `json:"text"`
	PageCount int    `json:"pageCount,omitempty"`
	CharCount int    `json:"charCount"`
}

// SaveDomainSourceMaterial 写入或覆盖某课的导入原文
func (s *Store) SaveDomainSourceMaterial(domainID string, mat DomainSourceMaterial) error {
	domainID = strings.TrimSpace(domainID)
	if domainID == "" {
		return fmt.Errorf("缺少领域 ID")
	}
	kind := strings.TrimSpace(mat.Kind)
	if kind == "" {
		return fmt.Errorf("缺少材料类型")
	}
	body := strings.TrimSpace(mat.Text)
	if body == "" {
		return fmt.Errorf("材料正文为空")
	}
	charCount := mat.CharCount
	if charCount <= 0 {
		charCount = len([]rune(body))
	}
	_, err := s.db.Exec(
		`INSERT INTO domain_source_materials (domain_id, kind, label, body, page_count, char_count)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(domain_id) DO UPDATE SET
		   kind = excluded.kind,
		   label = excluded.label,
		   body = excluded.body,
		   page_count = excluded.page_count,
		   char_count = excluded.char_count`,
		domainID, kind, strings.TrimSpace(mat.Label), body, mat.PageCount, charCount,
	)
	if err != nil {
		return fmt.Errorf("保存导入原文失败: %w", err)
	}
	return nil
}

// HasDomainSourceMaterial 是否存有导入原文
func (s *Store) HasDomainSourceMaterial(domainID string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM domain_source_materials WHERE domain_id = ?`, domainID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetDomainSourceMaterial 读取导入原文（需属于该用户）
func (s *Store) GetDomainSourceMaterial(userID, domainID string) (*DomainSourceMaterial, error) {
	userID = normalizeUserID(userID)
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("领域不存在")
	}
	var mat DomainSourceMaterial
	var label sql.NullString
	err = s.db.QueryRow(
		`SELECT domain_id, kind, label, body, page_count, char_count
		 FROM domain_source_materials WHERE domain_id = ?`,
		domainID,
	).Scan(&mat.DomainID, &mat.Kind, &label, &mat.Text, &mat.PageCount, &mat.CharCount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("该课程没有导入原文")
	}
	if err != nil {
		return nil, err
	}
	if label.Valid {
		mat.Label = label.String
	}
	return &mat, nil
}

// CopyDomainSourceMaterial 重建课程时把导入原文带到新课
func (s *Store) CopyDomainSourceMaterial(fromDomainID, toDomainID string) error {
	fromDomainID = strings.TrimSpace(fromDomainID)
	toDomainID = strings.TrimSpace(toDomainID)
	if fromDomainID == "" || toDomainID == "" || fromDomainID == toDomainID {
		return nil
	}
	res, err := s.db.Exec(
		`INSERT INTO domain_source_materials (domain_id, kind, label, body, page_count, char_count)
		 SELECT ?, kind, label, body, page_count, char_count
		 FROM domain_source_materials WHERE domain_id = ?
		 ON CONFLICT(domain_id) DO UPDATE SET
		   kind = excluded.kind,
		   label = excluded.label,
		   body = excluded.body,
		   page_count = excluded.page_count,
		   char_count = excluded.char_count`,
		toDomainID, fromDomainID,
	)
	if err != nil {
		return fmt.Errorf("复制导入原文失败: %w", err)
	}
	_, _ = res.RowsAffected()
	return nil
}
