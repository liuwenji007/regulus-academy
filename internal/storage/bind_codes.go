package storage

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const bindCodeTTL = 10 * time.Minute
const bindCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// CreateBindCode 为角色生成一次性 IM 绑定码（10 分钟有效）
func (s *Store) CreateBindCode(userID string) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, fmt.Errorf("无效的角色 ID")
	}
	if _, err := s.GetUser(userID); err != nil {
		return "", time.Time{}, err
	}
	code, err := randomBindCode(6)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(bindCodeTTL)
	now := time.Now().UTC().Format(time.RFC3339)
	expStr := expires.Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT INTO channel_bind_codes (code, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		code, userID, expStr, now,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成绑定码失败: %w", err)
	}
	return code, expires, nil
}

// RedeemBindCode 兑换绑定码，返回 user_id
func (s *Store) RedeemBindCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 {
		return "", fmt.Errorf("绑定码格式无效")
	}
	var userID, expiresAt string
	var usedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT user_id, expires_at, used_at FROM channel_bind_codes WHERE code = ?`, code,
	).Scan(&userID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("绑定码无效或已过期")
	}
	if err != nil {
		return "", err
	}
	if usedAt.Valid && usedAt.String != "" {
		return "", fmt.Errorf("绑定码已使用")
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		exp, _ = time.Parse(time.RFC3339Nano, expiresAt)
	}
	if exp.Before(time.Now().UTC()) {
		return "", fmt.Errorf("绑定码已过期，请在 Web 端重新生成")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE channel_bind_codes SET used_at = ? WHERE code = ? AND used_at IS NULL`, now, code,
	)
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", fmt.Errorf("绑定码已使用")
	}
	return userID, nil
}

func randomBindCode(n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(bindCodeChars)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = bindCodeChars[idx.Int64()]
	}
	return string(b), nil
}
