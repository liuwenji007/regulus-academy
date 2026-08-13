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
	ID                string     `json:"id"`
	DisplayName       string     `json:"displayName"`
	ProfileSummary    string     `json:"profileSummary,omitempty"`
	ProfileBackground string     `json:"profileBackground,omitempty"`
	ProfileGoal       string     `json:"profileGoal,omitempty"`
	ProfilePreference string     `json:"profilePreference,omitempty"`
	DomainProfiles    []UserDomainProfile `json:"domainProfiles,omitempty"`
	OnboardedAt       *time.Time `json:"onboardedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
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
		`SELECT id, COALESCE(display_name, ''), COALESCE(profile_summary, ''), COALESCE(profile_background, ''), COALESCE(profile_goal, ''), COALESCE(profile_preference, ''), onboarded_at, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		var onboarded sql.NullTime
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.ProfileSummary, &u.ProfileBackground, &u.ProfileGoal, &u.ProfilePreference, &onboarded, &u.CreatedAt); err != nil {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 单连接 SQLite 下不可在 rows 未关闭时嵌套查询，否则会死锁直到 busy_timeout。
	for i := range list {
		dps, err := s.ListDomainProfiles(list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].DomainProfiles = dps
	}
	return list, nil
}

// GetUser 获取单个角色
func (s *Store) GetUser(id string) (*User, error) {
	var u User
	var onboarded sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(display_name, ''), COALESCE(profile_summary, ''), COALESCE(profile_background, ''), COALESCE(profile_goal, ''), COALESCE(profile_preference, ''), onboarded_at, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.DisplayName, &u.ProfileSummary, &u.ProfileBackground, &u.ProfileGoal, &u.ProfilePreference, &onboarded, &u.CreatedAt)
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
	dps, _ := s.ListDomainProfiles(u.ID)
	u.DomainProfiles = dps
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

// DeleteUser 删除角色及其全部数据（课程、进度、会话、规划助手等）
func (s *Store) DeleteUser(id string) error {
	if id == "" {
		return fmt.Errorf("无效的角色 ID")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先清依赖会话 / 规划会话的子表，再清 user_id / domain_id 关联，最后删 domains 与 users
	for _, step := range []struct {
		q    string
		args []any
	}{
		{`DELETE FROM session_messages WHERE session_id IN (SELECT id FROM sessions WHERE user_id = ?)`, []any{id}},
		{`DELETE FROM planning_messages WHERE session_id IN (SELECT id FROM planning_sessions WHERE user_id = ?)`, []any{id}},
		{`DELETE FROM planning_sessions WHERE user_id = ?`, []any{id}},
		{`DELETE FROM channel_active_node WHERE user_id = ?`, []any{id}},
		{`DELETE FROM channel_bindings WHERE user_id = ?`, []any{id}},
		{`DELETE FROM channel_bind_codes WHERE user_id = ?`, []any{id}},
		{`DELETE FROM sessions WHERE user_id = ?`, []any{id}},
		{`DELETE FROM mistakes WHERE user_id = ?`, []any{id}},
		{`DELETE FROM user_progress WHERE user_id = ?`, []any{id}},
		{`DELETE FROM user_domain_profiles WHERE user_id = ?`, []any{id}},
		{`DELETE FROM user_domain_access WHERE user_id = ?`, []any{id}},
		{`DELETE FROM aside_messages WHERE user_id = ?`, []any{id}},
		{`DELETE FROM term_cards WHERE user_id = ?`, []any{id}},
		{`DELETE FROM knowledge_gaps WHERE user_id = ?`, []any{id}},
		{`DELETE FROM node_notes WHERE user_id = ?`, []any{id}},
		{`DELETE FROM domain_extensions WHERE user_id = ?`, []any{id}},
		{`DELETE FROM domain_build_jobs WHERE user_id = ?`, []any{id}},
		{`DELETE FROM user_llm_credentials WHERE user_id = ?`, []any{id}},
		{`DELETE FROM llm_usage_daily WHERE user_id = ?`, []any{id}},
		{`DELETE FROM llm_token_usage WHERE user_id = ?`, []any{id}},
		// domains 可能仍被其它用户行引用；本产品课程按 user_id 隔离，先清本用户域内残留
		{`DELETE FROM session_messages WHERE session_id IN (
			SELECT id FROM sessions WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)
		)`, []any{id}},
		{`DELETE FROM sessions WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM mistakes WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM user_progress WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM node_notes WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM user_domain_access WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM user_domain_profiles WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM domain_extensions WHERE domain_id IN (SELECT id FROM domains WHERE COALESCE(user_id, 'default') = ?)`, []any{id}},
		{`DELETE FROM domains WHERE COALESCE(user_id, 'default') = ?`, []any{id}},
		{`DELETE FROM users WHERE id = ?`, []any{id}},
	} {
		if _, err := tx.Exec(step.q, step.args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}
