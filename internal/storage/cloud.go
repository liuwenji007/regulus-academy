package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// UserLLMCredentials 用户 BYOK
type UserLLMCredentials struct {
	UserID           string
	Provider         string
	APIKeyEncrypted  string
	BaseURL          string
	Model            string
	UpdatedAt        string
}

// LLMUsageDaily 日用量
type LLMUsageDaily struct {
	MessageCount      int
	PromptTokens      int
	CompletionTokens  int
}

// AdminUserRow 管理员用户列表行
type AdminUserRow struct {
	User
	LastSeenAt       *time.Time `json:"lastSeenAt,omitempty"`
	MessagesToday    int        `json:"messagesToday"`
	TokensToday      int        `json:"tokensToday"`
	HasBYOK          bool       `json:"hasByok"`
}

func TodayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

func DaysAgoUTC(days int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -days)
}

func (s *Store) TouchUserLastSeen(userID string) error {
	userID = normalizeUserID(userID)
	if userID == "" || userID == DefaultUserID {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE users SET last_seen_at = ? WHERE id = ?`, now, userID)
	return err
}

func (s *Store) HasUserLLMCredentials(userID string) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(1) FROM user_llm_credentials WHERE user_id = ?`, normalizeUserID(userID),
	).Scan(&n)
	return n > 0, err
}

func (s *Store) GetUserLLMCredentials(userID string) (*UserLLMCredentials, error) {
	var c UserLLMCredentials
	err := s.db.QueryRow(
		`SELECT user_id, provider, api_key_encrypted, COALESCE(base_url,''), COALESCE(model,''), updated_at
		 FROM user_llm_credentials WHERE user_id = ?`, normalizeUserID(userID),
	).Scan(&c.UserID, &c.Provider, &c.APIKeyEncrypted, &c.BaseURL, &c.Model, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) SaveUserLLMCredentials(userID, provider, encKey, baseURL, model string) error {
	userID = normalizeUserID(userID)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO user_llm_credentials (user_id, provider, api_key_encrypted, base_url, model, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   provider = excluded.provider,
		   api_key_encrypted = excluded.api_key_encrypted,
		   base_url = excluded.base_url,
		   model = excluded.model,
		   updated_at = excluded.updated_at`,
		userID, provider, encKey, baseURL, model, now,
	)
	return err
}

func (s *Store) GetLLMUsageDaily(userID, date string) (LLMUsageDaily, error) {
	var u LLMUsageDaily
	err := s.db.QueryRow(
		`SELECT message_count, prompt_tokens, completion_tokens FROM llm_usage_daily WHERE user_id = ? AND usage_date = ?`,
		normalizeUserID(userID), date,
	).Scan(&u.MessageCount, &u.PromptTokens, &u.CompletionTokens)
	if err == sql.ErrNoRows {
		return LLMUsageDaily{}, nil
	}
	return u, err
}

func (s *Store) IncrementLLMMessageCount(userID, date string) error {
	userID = normalizeUserID(userID)
	_, err := s.db.Exec(
		`INSERT INTO llm_usage_daily (user_id, usage_date, message_count, prompt_tokens, completion_tokens)
		 VALUES (?, ?, 1, 0, 0)
		 ON CONFLICT(user_id, usage_date) DO UPDATE SET message_count = message_count + 1`,
		userID, date,
	)
	return err
}

func (s *Store) AddLLMUsageDailyTokens(userID, date string, prompt, completion int) error {
	userID = normalizeUserID(userID)
	_, err := s.db.Exec(
		`INSERT INTO llm_usage_daily (user_id, usage_date, message_count, prompt_tokens, completion_tokens)
		 VALUES (?, ?, 0, ?, ?)
		 ON CONFLICT(user_id, usage_date) DO UPDATE SET
		   prompt_tokens = prompt_tokens + excluded.prompt_tokens,
		   completion_tokens = completion_tokens + excluded.completion_tokens`,
		userID, date, prompt, completion,
	)
	return err
}

func (s *Store) AddLLMTokenUsage(userID, date, callKind, billedTo string, prompt, completion, total int) error {
	_, err := s.db.Exec(
		`INSERT INTO llm_token_usage (user_id, usage_date, prompt_tokens, completion_tokens, total_tokens, call_kind, billed_to)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		normalizeUserID(userID), date, prompt, completion, total, callKind, billedTo,
	)
	return err
}

func (s *Store) CountLearners(since time.Time) (total int, active int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id != ?`, DefaultUserID).Scan(&total); err != nil {
		return
	}
	sinceStr := since.Format(time.RFC3339)
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE id != ? AND last_seen_at IS NOT NULL AND last_seen_at >= ?`,
		DefaultUserID, sinceStr,
	).Scan(&active)
	return
}

func (s *Store) CountUsersCreatedOn(date string) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE id != ? AND date(created_at) = ?`,
		DefaultUserID, date,
	).Scan(&n)
	return n, err
}

func (s *Store) SumPlatformTokens(date string) (int, error) {
	var n sql.NullInt64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(total_tokens), 0) FROM llm_token_usage WHERE usage_date = ? AND billed_to = 'platform'`,
		date,
	).Scan(&n)
	if n.Valid {
		return int(n.Int64), err
	}
	return 0, err
}

func (s *Store) SumPlatformTokensTotal() (int, error) {
	var n sql.NullInt64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(total_tokens), 0) FROM llm_token_usage WHERE billed_to = 'platform'`,
	).Scan(&n)
	if n.Valid {
		return int(n.Int64), err
	}
	return 0, err
}

func (s *Store) CountRunningBuildJobs() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM domain_build_jobs WHERE status = 'running'`).Scan(&n)
	return n, err
}

func (s *Store) CountRunningBuildJobsForUser(userID string) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM domain_build_jobs WHERE user_id = ? AND status = 'running'`,
		normalizeUserID(userID),
	).Scan(&n)
	return n, err
}

func (s *Store) ListAdminUsers() ([]AdminUserRow, error) {
	date := TodayUTC()
	rows, err := s.db.Query(
		`SELECT u.id, COALESCE(u.display_name,''), COALESCE(u.profile_summary,''), u.onboarded_at, u.created_at, u.last_seen_at,
		        COALESCE(d.message_count, 0),
		        COALESCE(d.prompt_tokens, 0) + COALESCE(d.completion_tokens, 0),
		        CASE WHEN c.user_id IS NOT NULL THEN 1 ELSE 0 END
		 FROM users u
		 LEFT JOIN llm_usage_daily d ON d.user_id = u.id AND d.usage_date = ?
		 LEFT JOIN user_llm_credentials c ON c.user_id = u.id
		 WHERE u.id != ?
		 ORDER BY (u.last_seen_at IS NULL), u.last_seen_at DESC, u.created_at DESC`,
		date, DefaultUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AdminUserRow
	for rows.Next() {
		var row AdminUserRow
		var onboarded sql.NullTime
		var lastSeen sql.NullString
		var hasBYOK int
		if err := rows.Scan(
			&row.ID, &row.DisplayName, &row.ProfileSummary, &onboarded, &row.CreatedAt, &lastSeen,
			&row.MessagesToday, &row.TokensToday, &hasBYOK,
		); err != nil {
			return nil, err
		}
		if onboarded.Valid {
			t := onboarded.Time
			row.OnboardedAt = &t
		}
		if lastSeen.Valid && lastSeen.String != "" {
			if t, e := time.Parse(time.RFC3339, lastSeen.String); e == nil {
				row.LastSeenAt = &t
			}
		}
		if row.DisplayName == "" {
			row.DisplayName = "未命名"
		}
		row.HasBYOK = hasBYOK != 0
		list = append(list, row)
	}
	return list, rows.Err()
}

// AdminUsageByDay 按日聚合平台 token
func (s *Store) AdminUsageByDay(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 14
	}
	rows, err := s.db.Query(
		`SELECT usage_date,
		        SUM(CASE WHEN billed_to = 'platform' THEN total_tokens ELSE 0 END) as platform_tokens,
		        SUM(CASE WHEN billed_to = 'byok' THEN total_tokens ELSE 0 END) as byok_tokens
		 FROM llm_token_usage
		 GROUP BY usage_date
		 ORDER BY usage_date DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var date string
		var platform, byok int
		if err := rows.Scan(&date, &platform, &byok); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"date": date, "platformTokens": platform, "byokTokens": byok,
		})
	}
	return out, rows.Err()
}

func (s *Store) ResetUserDailyQuota(userID, date string) error {
	res, err := s.db.Exec(
		`DELETE FROM llm_usage_daily WHERE user_id = ? AND usage_date = ?`,
		normalizeUserID(userID), date,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("无当日用量记录")
	}
	return nil
}
