package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// User 本地学习角色
type User struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// EnsureUser 确保用户记录存在（不覆盖已有显示名）
func (s *Store) EnsureUser(id string) error {
	if id == "" {
		id = DefaultUserID
	}
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO users (id, display_name, created_at) VALUES (?, ?, ?)`,
		id, "", time.Now().UTC(),
	)
	return err
}

// CreateUser 创建新学习角色
func (s *Store) CreateUser(displayName string) (*User, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return nil, fmt.Errorf("姓名不能为空")
	}
	if utf8.RuneCountInString(name) > 32 {
		return nil, fmt.Errorf("姓名不能超过 32 个字符")
	}
	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`INSERT INTO users (id, display_name, created_at) VALUES (?, ?, ?)`,
		id, name, now,
	)
	if err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}
	return &User{ID: id, DisplayName: name, CreatedAt: now}, nil
}

// ListUsers 列出全部学习角色
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(
		`SELECT id, COALESCE(display_name, ''), created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.CreatedAt); err != nil {
			return nil, err
		}
		if u.DisplayName == "" {
			u.DisplayName = "未命名"
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// GetUser 获取单个角色
func (s *Store) GetUser(id string) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, COALESCE(display_name, ''), created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.DisplayName, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("角色不存在")
	}
	if err != nil {
		return nil, err
	}
	if u.DisplayName == "" {
		u.DisplayName = "未命名"
	}
	return &u, nil
}

// DeleteUser 删除角色及其全部数据（课程、进度、会话等）
func (s *Store) DeleteUser(id string) error {
	if id == "" {
		return fmt.Errorf("无效的角色 ID")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`DELETE FROM session_messages WHERE session_id IN (SELECT id FROM sessions WHERE user_id = ?)`, id,
	); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM channel_active_node WHERE user_id = ?`,
		`DELETE FROM channel_bindings WHERE user_id = ?`,
		`DELETE FROM sessions WHERE user_id = ?`,
		`DELETE FROM mistakes WHERE user_id = ?`,
		`DELETE FROM user_progress WHERE user_id = ?`,
		`DELETE FROM domains WHERE COALESCE(user_id, 'default') = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
