package storage

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultMaxUserSourceBytes = 32 * 1024 * 1024

// DomainSource 一门课挂载的原文。
type DomainSource struct {
	ID        string
	UserID    string
	DomainID  string
	Kind      string
	Filename  string
	URL       string
	Text      string
	ByteSize  int
	CreatedAt time.Time
}

// DomainSourceMeta 列表用，不含正文。
type DomainSourceMeta struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Filename   string    `json:"filename,omitempty"`
	URL        string    `json:"url,omitempty"`
	ByteSize   int       `json:"byteSize"`
	ChunkCount int       `json:"chunkCount"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SourceChunk 召回用小块。
type SourceChunk struct {
	ID         string
	SourceID   string
	DomainID   string
	ChunkIndex int
	Text       string
	NodeKey    string
	Filename   string
	URL        string
	Kind       string
}

// MaxUserSourceBytes 每用户资料总量上限。
func MaxUserSourceBytes() int {
	raw := strings.TrimSpace(os.Getenv("REGULUS_SOURCE_MAX_USER_BYTES"))
	if raw == "" {
		return defaultMaxUserSourceBytes
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultMaxUserSourceBytes
	}
	return n
}

// UserSourceBytes 当前用户已存资料字节数。
func (s *Store) UserSourceBytes(userID string) (int, error) {
	var n sql.NullInt64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(byte_size), 0) FROM domain_sources WHERE user_id = ?`,
		userID,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return int(n.Int64), nil
}

// AssertSourceBudget 追加前检查总量。
func (s *Store) AssertSourceBudget(userID string, addBytes int) error {
	if addBytes < 0 {
		addBytes = 0
	}
	used, err := s.UserSourceBytes(userID)
	if err != nil {
		return err
	}
	limit := MaxUserSourceBytes()
	if used+addBytes > limit {
		return fmt.Errorf("课程资料总量超过上限（%d MB）", limit/(1024*1024))
	}
	return nil
}

// InsertDomainSource 写入原文并切好的小块。chunks 为空时仍保存原文。
func (s *Store) InsertDomainSource(src *DomainSource, chunks []string) (*DomainSource, error) {
	if src == nil {
		return nil, fmt.Errorf("资料为空")
	}
	text := strings.TrimSpace(src.Text)
	if text == "" {
		return nil, fmt.Errorf("资料正文为空")
	}
	if err := s.AssertSourceBudget(src.UserID, len([]byte(text))); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(src.ID)
	if id == "" {
		id = uuid.NewString()
	}
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO domain_sources (id, user_id, domain_id, kind, filename, url, text, byte_size, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, src.UserID, src.DomainID, src.Kind, src.Filename, src.URL, text, len([]byte(text)), now,
	); err != nil {
		return nil, err
	}
	for i, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO source_chunks (id, source_id, domain_id, chunk_index, text, node_key, created_at)
			 VALUES (?, ?, ?, ?, ?, '', ?)`,
			uuid.NewString(), id, src.DomainID, i, chunk, now,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := *src
	out.ID = id
	out.Text = text
	out.ByteSize = len([]byte(text))
	out.CreatedAt = now
	return &out, nil
}

// ListDomainSourceMeta 列出一门课的资料，不含正文。
func (s *Store) ListDomainSourceMeta(userID, domainID string) ([]DomainSourceMeta, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.kind, s.filename, s.url, s.byte_size, s.created_at,
		        (SELECT COUNT(*) FROM source_chunks c WHERE c.source_id = s.id)
		 FROM domain_sources s
		 WHERE s.user_id = ? AND s.domain_id = ?
		 ORDER BY s.created_at ASC`,
		userID, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DomainSourceMeta
	for rows.Next() {
		var m DomainSourceMeta
		if err := rows.Scan(&m.ID, &m.Kind, &m.Filename, &m.URL, &m.ByteSize, &m.CreatedAt, &m.ChunkCount); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListDomainSourceTexts 取原文，供重建课程时带上参考摘录。
func (s *Store) ListDomainSourceTexts(userID, domainID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT text FROM domain_sources WHERE user_id = ? AND domain_id = ? ORDER BY created_at ASC`,
		userID, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		if strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out, rows.Err()
}

// ListSourceChunks 一门课的全部小块（含来源文件名）。
func (s *Store) ListSourceChunks(userID, domainID string) ([]SourceChunk, error) {
	rows, err := s.db.Query(
		`SELECT c.id, c.source_id, c.domain_id, c.chunk_index, c.text, c.node_key,
		        s.filename, s.url, s.kind
		 FROM source_chunks c
		 JOIN domain_sources s ON s.id = c.source_id
		 WHERE s.user_id = ? AND c.domain_id = ?
		 ORDER BY s.created_at ASC, c.chunk_index ASC`,
		userID, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SourceChunk
	for rows.Next() {
		var c SourceChunk
		if err := rows.Scan(
			&c.ID, &c.SourceID, &c.DomainID, &c.ChunkIndex, &c.Text, &c.NodeKey,
			&c.Filename, &c.URL, &c.Kind,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// HasDomainSources 课程是否挂过用户资料。
func (s *Store) HasDomainSources(userID, domainID string) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM domain_sources WHERE user_id = ? AND domain_id = ?`,
		userID, domainID,
	).Scan(&n)
	return n > 0, err
}

// ApplyChunkBindings 清空后写回 chunk → node_key。未出现在 bindings 中的块保持空。
func (s *Store) ApplyChunkBindings(userID, domainID string, bindings map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE source_chunks SET node_key = ''
		 WHERE domain_id = ? AND source_id IN (
		   SELECT id FROM domain_sources WHERE user_id = ? AND domain_id = ?
		 )`,
		domainID, userID, domainID,
	); err != nil {
		return err
	}
	for id, nodeKey := range bindings {
		id = strings.TrimSpace(id)
		nodeKey = strings.TrimSpace(nodeKey)
		if id == "" || nodeKey == "" {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE source_chunks SET node_key = ?
			 WHERE id = ? AND domain_id = ? AND source_id IN (
			   SELECT id FROM domain_sources WHERE user_id = ? AND domain_id = ?
			 )`,
			nodeKey, id, domainID, userID, domainID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MoveDomainSources 把资料挂到新课程并清空节点绑定，便于重新接地。
func (s *Store) MoveDomainSources(userID, fromDomainID, toDomainID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE source_chunks SET domain_id = ?, node_key = ''
		 WHERE domain_id = ? AND source_id IN (
		   SELECT id FROM domain_sources WHERE user_id = ? AND domain_id = ?
		 )`,
		toDomainID, fromDomainID, userID, fromDomainID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE domain_sources SET domain_id = ? WHERE user_id = ? AND domain_id = ?`,
		toDomainID, userID, fromDomainID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateDomainNodesJSON 只更新节点边界，不改知识树版本。
func (s *Store) UpdateDomainNodesJSON(userID, domainID, nodesJSON string) error {
	if strings.TrimSpace(nodesJSON) == "" {
		nodesJSON = "{}"
	}
	res, err := s.db.Exec(
		`UPDATE domains SET nodes_json = ? WHERE id = ? AND COALESCE(user_id, 'default') = ?`,
		nodesJSON, domainID, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("领域不存在")
	}
	return nil
}

// DeleteTermCardsForDomain 资料变更后失效术语卡缓存。
func (s *Store) DeleteTermCardsForDomain(userID, domainID string) error {
	_, err := s.db.Exec(
		`DELETE FROM term_cards WHERE user_id = ? AND domain_id = ?`,
		userID, domainID,
	)
	return err
}
