package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// ChannelBinding IM 平台用户与学习角色的绑定
type ChannelBinding struct {
	Platform         string    `json:"platform"`
	PlatformUserID   string    `json:"platformUserId"`
	UserID           string    `json:"userId"`
	DisplayNameSnap  string    `json:"displayNameSnapshot,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ChannelActiveNode 用户当前 IM 学习上下文
type ChannelActiveNode struct {
	UserID    string    `json:"userId"`
	DomainID  string    `json:"domainId"`
	NodeKey   string    `json:"nodeKey"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetChannelBinding 查询平台用户绑定
func (s *Store) GetChannelBinding(platform, platformUserID string) (*ChannelBinding, error) {
	var b ChannelBinding
	var createdAt string
	err := s.db.QueryRow(
		`SELECT platform, platform_user_id, user_id, COALESCE(display_name_snapshot, ''), created_at
		 FROM channel_bindings WHERE platform = ? AND platform_user_id = ?`,
		platform, platformUserID,
	).Scan(&b.Platform, &b.PlatformUserID, &b.UserID, &b.DisplayNameSnap, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if b.CreatedAt.IsZero() {
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}
	return &b, nil
}

// UpsertChannelBinding 创建或更新绑定
func (s *Store) UpsertChannelBinding(platform, platformUserID, userID, displayName string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO channel_bindings (platform, platform_user_id, user_id, display_name_snapshot, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(platform, platform_user_id) DO UPDATE SET
		   user_id = excluded.user_id,
		   display_name_snapshot = excluded.display_name_snapshot`,
		platform, platformUserID, userID, displayName, now,
	)
	return err
}

// DeleteChannelBindingsForUser 删除用户的全部 channel 绑定
func (s *Store) DeleteChannelBindingsForUser(userID string) error {
	_, err := s.db.Exec(`DELETE FROM channel_bindings WHERE user_id = ?`, userID)
	return err
}

// ListChannelBindingsForUser 列出角色的 IM 绑定
func (s *Store) ListChannelBindingsForUser(userID string) ([]ChannelBinding, error) {
	rows, err := s.db.Query(
		`SELECT platform, platform_user_id, user_id, COALESCE(display_name_snapshot, ''), created_at
		 FROM channel_bindings WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ChannelBinding
	for rows.Next() {
		var b ChannelBinding
		var createdAt string
		if err := rows.Scan(&b.Platform, &b.PlatformUserID, &b.UserID, &b.DisplayNameSnap, &createdAt); err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if b.CreatedAt.IsZero() {
			b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// FindUserByDisplayName 按显示名精确查找角色
func (s *Store) FindUserByDisplayName(displayName string) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, COALESCE(display_name, ''), created_at FROM users WHERE display_name = ?`,
		displayName,
	).Scan(&u.ID, &u.DisplayName, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("未找到名为「%s」的角色，请先在 Web 端创建", displayName)
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetChannelActiveNode 记录用户当前学习的课程节点
func (s *Store) SetChannelActiveNode(userID, domainID, nodeKey string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO channel_active_node (user_id, domain_id, node_key, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   domain_id = excluded.domain_id,
		   node_key = excluded.node_key,
		   updated_at = excluded.updated_at`,
		userID, domainID, nodeKey, now,
	)
	return err
}

// GetChannelActiveNode 获取用户当前 IM 学习节点
func (s *Store) GetChannelActiveNode(userID string) (*ChannelActiveNode, error) {
	var n ChannelActiveNode
	var updatedAt string
	err := s.db.QueryRow(
		`SELECT user_id, domain_id, node_key, updated_at FROM channel_active_node WHERE user_id = ?`,
		userID,
	).Scan(&n.UserID, &n.DomainID, &n.NodeKey, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	}
	return &n, nil
}

// DeleteChannelActiveNode 删除 IM 学习上下文
func (s *Store) DeleteChannelActiveNode(userID string) error {
	_, err := s.db.Exec(`DELETE FROM channel_active_node WHERE user_id = ?`, userID)
	return err
}
