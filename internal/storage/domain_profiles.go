package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// UserDomainProfile 按课程的学习方向摘要
type UserDomainProfile struct {
	UserID    string    `json:"userId"`
	DomainID  string    `json:"domainId"`
	DomainName string   `json:"domainName,omitempty"`
	Summary   string    `json:"summary"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const maxDomainProfileSummaryRunes = 200

// GetDomainProfile 读取单课摘要
func (s *Store) GetDomainProfile(userID, domainID string) (*UserDomainProfile, error) {
	userID = normalizeUserID(userID)
	var row UserDomainProfile
	var updated string
	err := s.db.QueryRow(
		`SELECT user_id, domain_id, COALESCE(summary,''), updated_at FROM user_domain_profiles WHERE user_id = ? AND domain_id = ?`,
		userID, domainID,
	).Scan(&row.UserID, &row.DomainID, &row.Summary, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if t, e := time.Parse(time.RFC3339, updated); e == nil {
		row.UpdatedAt = t
	}
	return &row, nil
}

// ListDomainProfiles 列出用户全部按课摘要（含课程名）
func (s *Store) ListDomainProfiles(userID string) ([]UserDomainProfile, error) {
	userID = normalizeUserID(userID)
	rows, err := s.db.Query(
		`SELECT udp.user_id, udp.domain_id, COALESCE(d.name,''), COALESCE(udp.summary,''), udp.updated_at
		 FROM user_domain_profiles udp
		 LEFT JOIN domains d ON d.id = udp.domain_id
		 WHERE udp.user_id = ?
		 ORDER BY udp.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserDomainProfile
	for rows.Next() {
		var row UserDomainProfile
		var updated string
		if err := rows.Scan(&row.UserID, &row.DomainID, &row.DomainName, &row.Summary, &updated); err != nil {
			return nil, err
		}
		if t, e := time.Parse(time.RFC3339, updated); e == nil {
			row.UpdatedAt = t
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// UpsertDomainProfile 写入按课摘要；completedAt 较旧时丢弃（乱序保护）
func (s *Store) UpsertDomainProfile(userID, domainID, summary string, completedAt time.Time) error {
	userID = normalizeUserID(userID)
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	if runeLen(summary) > maxDomainProfileSummaryRunes {
		summary = truncateRunesStorage(summary, maxDomainProfileSummaryRunes)
	}
	ok, err := s.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	ts := completedAt.UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT INTO user_domain_profiles (user_id, domain_id, summary, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, domain_id) DO UPDATE SET
		   summary = excluded.summary,
		   updated_at = excluded.updated_at
		 WHERE excluded.updated_at > user_domain_profiles.updated_at`,
		userID, domainID, summary, ts,
	)
	return err
}

// DeleteDomainProfilesForDomain 删课级联
func (s *Store) DeleteDomainProfilesForDomain(userID, domainID string) error {
	userID = normalizeUserID(userID)
	_, err := s.db.Exec(`DELETE FROM user_domain_profiles WHERE user_id = ? AND domain_id = ?`, userID, domainID)
	return err
}

// DeleteDomainProfilesForUser 删角色级联
func (s *Store) DeleteDomainProfilesForUser(userID string) error {
	if userID == "" {
		return fmt.Errorf("无效的角色 ID")
	}
	_, err := s.db.Exec(`DELETE FROM user_domain_profiles WHERE user_id = ?`, userID)
	return err
}

// ListDomainIDsWithProgress 返回用户有学习进度的课程 ID
func (s *Store) ListDomainIDsWithProgress(userID string) ([]string, error) {
	userID = normalizeUserID(userID)
	rows, err := s.db.Query(
		`SELECT DISTINCT domain_id FROM user_progress WHERE user_id = ? ORDER BY domain_id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func runeLen(s string) int {
	return len([]rune(s))
}

func truncateRunesStorage(s string, max int) string {
	if max <= 0 || runeLen(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
