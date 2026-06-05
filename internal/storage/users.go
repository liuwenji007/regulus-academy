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
	ID              string     `json:"id"`
	DisplayName     string     `json:"displayName"`
	ProfileSummary  string     `json:"profileSummary,omitempty"`
	OnboardedAt     *time.Time `json:"onboardedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// NeedsOnboarding 是否尚未完成冷启动引导。
func NeedsOnboarding(u *User) bool {
	return u != nil && u.OnboardedAt == nil
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
		`SELECT id, COALESCE(display_name, ''), COALESCE(profile_summary, ''), onboarded_at, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		var onboarded sql.NullTime
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.ProfileSummary, &onboarded, &u.CreatedAt); err != nil {
			return nil, err
		}
		if onboarded.Valid {
			t := onboarded.Time
			u.OnboardedAt = &t
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
	var onboarded sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(display_name, ''), COALESCE(profile_summary, ''), onboarded_at, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.DisplayName, &u.ProfileSummary, &onboarded, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("角色不存在")
	}
	if err != nil {
		return nil, err
	}
	if onboarded.Valid {
		t := onboarded.Time
		u.OnboardedAt = &t
	}
	if u.DisplayName == "" {
		u.DisplayName = "未命名"
	}
	return &u, nil
}

// MarkUserOnboarded 标记用户已完成冷启动引导。
func (s *Store) MarkUserOnboarded(userID string) error {
	if userID == "" {
		return fmt.Errorf("无效的角色 ID")
	}
	now := time.Now().UTC()
	res, err := s.db.Exec(`UPDATE users SET onboarded_at = ? WHERE id = ?`, now, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("角色不存在")
	}
	return nil
}

const maxProfileSummaryRunes = 500

// UpdateUserProfileSummary 更新有界用户画像（≤500 字）
func (s *Store) UpdateUserProfileSummary(userID, summary string) error {
	summary = strings.TrimSpace(summary)
	if utf8.RuneCountInString(summary) > maxProfileSummaryRunes {
		return fmt.Errorf("用户画像不能超过 %d 字", maxProfileSummaryRunes)
	}
	res, err := s.db.Exec(`UPDATE users SET profile_summary = ? WHERE id = ?`, summary, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("角色不存在")
	}
	return nil
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
