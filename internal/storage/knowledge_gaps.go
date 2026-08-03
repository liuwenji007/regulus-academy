package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GapSource 常量
const (
	GapSourceAsideLookup = "aside_lookup"
	GapSourceMistake     = "mistake"
	GapSourceCoachGap    = "coach_gap"
	GapSourceExplicit    = "explicit"
)

// UpsertKnowledgeGap 写入或累加缺口；concept 应已归一化
func (s *Store) UpsertKnowledgeGap(gap *KnowledgeGap) (*KnowledgeGap, error) {
	if gap == nil {
		return nil, fmt.Errorf("缺口为空")
	}
	concept := strings.TrimSpace(gap.Concept)
	if concept == "" {
		return nil, fmt.Errorf("概念为空")
	}
	source := strings.TrimSpace(gap.Source)
	if source == "" {
		source = GapSourceAsideLookup
	}
	now := time.Now().UTC()
	severity := gap.Severity
	if severity <= 0 {
		severity = defaultGapSeverity(source)
	}

	var existing KnowledgeGap
	var resolved sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, concept, source, hit_count, severity,
		        matched_domain_id, matched_node_key, reason, resolved_at, last_hit_at, created_at
		 FROM knowledge_gaps
		 WHERE user_id = ? AND domain_id = ? AND concept = ? AND source = ?`,
		gap.UserID, gap.DomainID, concept, source,
	).Scan(
		&existing.ID, &existing.UserID, &existing.DomainID, &existing.NodeKey,
		&existing.Concept, &existing.Source, &existing.HitCount, &existing.Severity,
		&existing.MatchedDomainID, &existing.MatchedNodeKey, &existing.Reason,
		&resolved, &existing.LastHitAt, &existing.CreatedAt,
	)
	if err == sql.ErrNoRows {
		res, err := s.db.Exec(
			`INSERT INTO knowledge_gaps
			 (user_id, domain_id, node_key, concept, source, hit_count, severity,
			  matched_domain_id, matched_node_key, reason, last_hit_at, created_at)
			 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)`,
			gap.UserID, gap.DomainID, gap.NodeKey, concept, source, severity,
			gap.MatchedDomainID, gap.MatchedNodeKey, gap.Reason, now, now,
		)
		if err != nil {
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		out := *gap
		out.ID = id
		out.Concept = concept
		out.Source = source
		out.HitCount = 1
		out.Severity = severity
		out.LastHitAt = now
		out.CreatedAt = now
		return &out, nil
	}
	if err != nil {
		return nil, err
	}
	if resolved.Valid {
		t := resolved.Time
		existing.ResolvedAt = &t
	}

	newSeverity := existing.Severity + severity*0.15
	if newSeverity > 10 {
		newSeverity = 10
	}
	matchedDomain := gap.MatchedDomainID
	if matchedDomain == "" {
		matchedDomain = existing.MatchedDomainID
	}
	matchedNode := gap.MatchedNodeKey
	if matchedNode == "" {
		matchedNode = existing.MatchedNodeKey
	}
	reason := gap.Reason
	if reason == "" {
		reason = existing.Reason
	}
	nodeKey := gap.NodeKey
	if nodeKey == "" {
		nodeKey = existing.NodeKey
	}
	_, err = s.db.Exec(
		`UPDATE knowledge_gaps SET hit_count = hit_count + 1, severity = ?, last_hit_at = ?,
		 resolved_at = NULL, matched_domain_id = ?, matched_node_key = ?, reason = ?, node_key = ?
		 WHERE id = ?`,
		newSeverity, now, matchedDomain, matchedNode, reason, nodeKey, existing.ID,
	)
	if err != nil {
		return nil, err
	}
	existing.HitCount++
	existing.Severity = newSeverity
	existing.LastHitAt = now
	existing.ResolvedAt = nil
	existing.MatchedDomainID = matchedDomain
	existing.MatchedNodeKey = matchedNode
	existing.Reason = reason
	existing.NodeKey = nodeKey
	return &existing, nil
}

func defaultGapSeverity(source string) float64 {
	switch source {
	case GapSourceMistake:
		return 2.0
	case GapSourceCoachGap:
		return 1.8
	case GapSourceExplicit:
		return 1.5
	default:
		return 1.0
	}
}

// ListOpenKnowledgeGaps 未关闭缺口，按 severity + 最近命中排序
func (s *Store) ListOpenKnowledgeGaps(userID, domainID string, limit int) ([]KnowledgeGap, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(domainID) == "" {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, concept, source, hit_count, severity,
			        matched_domain_id, matched_node_key, reason, resolved_at, last_hit_at, created_at
			 FROM knowledge_gaps
			 WHERE user_id = ? AND resolved_at IS NULL
			 ORDER BY severity DESC, last_hit_at DESC LIMIT ?`,
			userID, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, user_id, domain_id, node_key, concept, source, hit_count, severity,
			        matched_domain_id, matched_node_key, reason, resolved_at, last_hit_at, created_at
			 FROM knowledge_gaps
			 WHERE user_id = ? AND domain_id = ? AND resolved_at IS NULL
			 ORDER BY severity DESC, last_hit_at DESC LIMIT ?`,
			userID, domainID, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeGaps(rows)
}

// ResolveKnowledgeGap 手动或学完后关闭缺口
func (s *Store) ResolveKnowledgeGap(userID string, id int64) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`UPDATE knowledge_gaps SET resolved_at = ? WHERE id = ? AND user_id = ? AND resolved_at IS NULL`,
		now, id, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("缺口不存在或已关闭")
	}
	return nil
}

// ResolveKnowledgeGapsByNode 学完某节点后关闭「映射到该节点」的缺口。
// 只认 matched_*，避免把「在当前节点发现的前置缺口」误关。
func (s *Store) ResolveKnowledgeGapsByNode(userID, domainID, nodeKey string) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE knowledge_gaps SET resolved_at = ?
		 WHERE user_id = ? AND resolved_at IS NULL
		   AND matched_domain_id = ? AND matched_node_key = ?`,
		now, userID, domainID, nodeKey,
	)
	return err
}

// UpdateKnowledgeGapMatch 缓存概念→节点映射（须带 user_id 防串改）
func (s *Store) UpdateKnowledgeGapMatch(userID string, id int64, matchedDomainID, matchedNodeKey string) error {
	_, err := s.db.Exec(
		`UPDATE knowledge_gaps SET matched_domain_id = ?, matched_node_key = ?
		 WHERE id = ? AND user_id = ?`,
		matchedDomainID, matchedNodeKey, id, userID,
	)
	return err
}

func scanKnowledgeGaps(rows *sql.Rows) ([]KnowledgeGap, error) {
	var list []KnowledgeGap
	for rows.Next() {
		var g KnowledgeGap
		var resolved sql.NullTime
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.DomainID, &g.NodeKey, &g.Concept, &g.Source,
			&g.HitCount, &g.Severity, &g.MatchedDomainID, &g.MatchedNodeKey,
			&g.Reason, &resolved, &g.LastHitAt, &g.CreatedAt,
		); err != nil {
			return nil, err
		}
		if resolved.Valid {
			t := resolved.Time
			g.ResolvedAt = &t
		}
		list = append(list, g)
	}
	return list, rows.Err()
}
