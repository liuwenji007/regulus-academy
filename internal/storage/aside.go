package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AddAsideMessage 写入旁路消息
func (s *Store) AddAsideMessage(msg *AsideMessage) (*AsideMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("消息为空")
	}
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO aside_messages
		 (user_id, domain_id, node_key, coach_session_id, role, content, anchor_text, intent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.UserID, msg.DomainID, msg.NodeKey, msg.CoachSessionID,
		msg.Role, msg.Content, msg.AnchorText, msg.Intent, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	out := *msg
	out.ID = id
	out.CreatedAt = now
	return &out, nil
}

// ListAsideMessages 按用户列出旁路消息（升序）；domainID 为空则不按课过滤
func (s *Store) ListAsideMessages(userID, domainID string, limit int) ([]AsideMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(domainID) == "" {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, coach_session_id, role, content, anchor_text, intent, created_at
			 FROM aside_messages
			 WHERE user_id = ?
			 ORDER BY id DESC LIMIT ?`,
			userID, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, coach_session_id, role, content, anchor_text, intent, created_at
			 FROM aside_messages
			 WHERE user_id = ? AND domain_id = ?
			 ORDER BY id DESC LIMIT ?`,
			userID, domainID, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AsideMessage
	for rows.Next() {
		var m AsideMessage
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.DomainID, &m.NodeKey, &m.CoachSessionID,
			&m.Role, &m.Content, &m.AnchorText, &m.Intent, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 逆序为升序
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

// GetTermCard 按归一化术语取卡片
func (s *Store) GetTermCard(userID, domainID, normalizedTerm string) (*TermCard, error) {
	var c TermCard
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, normalized_term, original_text, card_json, hit_count, last_hit_at, created_at
		 FROM term_cards WHERE user_id = ? AND domain_id = ? AND normalized_term = ?`,
		userID, domainID, normalizedTerm,
	).Scan(
		&c.ID, &c.UserID, &c.DomainID, &c.NodeKey, &c.NormalizedTerm, &c.OriginalText,
		&c.CardJSON, &c.HitCount, &c.LastHitAt, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertTermCard 写入或命中计数+1（原子 upsert，避免并发首次撞 UNIQUE）
func (s *Store) UpsertTermCard(card *TermCard) (*TermCard, error) {
	if card == nil {
		return nil, fmt.Errorf("卡片为空")
	}
	now := time.Now().UTC()
	cardJSON := strings.TrimSpace(card.CardJSON)
	if cardJSON == "" {
		cardJSON = "{}"
	}
	_, err := s.db.Exec(
		`INSERT INTO term_cards
		 (user_id, domain_id, node_key, normalized_term, original_text, card_json, hit_count, last_hit_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
		 ON CONFLICT(user_id, domain_id, normalized_term) DO UPDATE SET
		   hit_count = hit_count + 1,
		   last_hit_at = excluded.last_hit_at,
		   card_json = CASE
		     WHEN excluded.card_json IS NOT NULL AND excluded.card_json != '' AND excluded.card_json != '{}'
		     THEN excluded.card_json ELSE card_json END,
		   node_key = CASE WHEN excluded.node_key != '' THEN excluded.node_key ELSE node_key END,
		   original_text = CASE WHEN excluded.original_text != '' THEN excluded.original_text ELSE original_text END`,
		card.UserID, card.DomainID, card.NodeKey, card.NormalizedTerm,
		card.OriginalText, cardJSON, now, now,
	)
	if err != nil {
		return nil, err
	}
	return s.GetTermCard(card.UserID, card.DomainID, card.NormalizedTerm)
}

// ListTermCards 术语本列表
func (s *Store) ListTermCards(userID, domainID string, limit int) ([]TermCard, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(domainID) == "" {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, normalized_term, original_text, card_json, hit_count, last_hit_at, created_at
			 FROM term_cards WHERE user_id = ?
			 ORDER BY last_hit_at DESC LIMIT ?`,
			userID, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, normalized_term, original_text, card_json, hit_count, last_hit_at, created_at
			 FROM term_cards WHERE user_id = ? AND domain_id = ?
			 ORDER BY last_hit_at DESC LIMIT ?`,
			userID, domainID, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []TermCard
	for rows.Next() {
		var c TermCard
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.DomainID, &c.NodeKey, &c.NormalizedTerm, &c.OriginalText,
			&c.CardJSON, &c.HitCount, &c.LastHitAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
