package storage

import (
	"database/sql"
	"time"
)

// DomainAccess 用户最近打开某门课的记录（进树 / 开练都会刷新）
type DomainAccess struct {
	UserID     string
	DomainID   string
	NodeKey    string
	AccessedAt time.Time
}

// TouchDomainAccess 记录用户进入某门课（侧栏「上一节」用）
func (s *Store) TouchDomainAccess(userID, domainID, nodeKey string) error {
	userID = normalizeUserID(userID)
	if domainID == "" {
		return nil
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`INSERT INTO user_domain_access (user_id, domain_id, node_key, accessed_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, domain_id) DO UPDATE SET
		   node_key = CASE WHEN excluded.node_key != '' THEN excluded.node_key ELSE user_domain_access.node_key END,
		   accessed_at = excluded.accessed_at`,
		userID, domainID, nodeKey, now,
	)
	return err
}

// FindLastDomainAccess 最近进入的一门课
func (s *Store) FindLastDomainAccess(userID string) (*DomainAccess, error) {
	userID = normalizeUserID(userID)
	var a DomainAccess
	var accessedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT user_id, domain_id, COALESCE(node_key,''), accessed_at
		 FROM user_domain_access
		 WHERE user_id = ?
		 ORDER BY accessed_at DESC LIMIT 1`,
		userID,
	).Scan(&a.UserID, &a.DomainID, &a.NodeKey, &accessedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if accessedAt.Valid {
		a.AccessedAt = accessedAt.Time
	}
	return &a, nil
}

// TouchSession 刷新会话活动时间（恢复旧会话时也算「正在学」）
func (s *Store) TouchSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	return err
}

// FindLatestSessionInDomain 该课程最近一次会话（任意节点）
func (s *Store) FindLatestSessionInDomain(userID, domainID string) (*Session, error) {
	userID = normalizeUserID(userID)
	var sess Session
	var updatedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, status, created_at,
		 COALESCE(phase,'explain'), COALESCE(context_json,'{}'), COALESCE(domain_slug,''),
		 updated_at
		 FROM sessions
		 WHERE user_id = ? AND domain_id = ?
		 ORDER BY COALESCE(updated_at, created_at) DESC LIMIT 1`,
		userID, domainID,
	).Scan(&sess.ID, &sess.UserID, &sess.DomainID, &sess.NodeKey, &sess.Status, &sess.CreatedAt,
		&sess.Phase, &sess.ContextJSON, &sess.DomainSlug, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	applySessionUpdatedAt(&sess, updatedAt)
	return &sess, nil
}
