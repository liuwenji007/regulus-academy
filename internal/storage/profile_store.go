package storage

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxGlobalBackgroundRunes  = 200
	maxGlobalGoalRunes        = 150
	maxGlobalPreferenceRunes  = 100
	maxComposedGlobalRunes    = 300
)

// ParseBackgroundGoal 从 legacy profile_summary 剥离【进展】段，仅保留背景/目标信息。
func ParseBackgroundGoal(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}
	if idx := strings.Index(summary, "【进展】"); idx >= 0 {
		summary = strings.TrimSpace(summary[:idx])
	}
	if idx := strings.Index(summary, "【目标】"); idx >= 0 {
		summary = strings.TrimSpace(summary[:idx])
	}
	return strings.TrimSpace(summary)
}

// StripProgressSection 移除【进展】/【目标】段（merge 护栏）
func StripProgressSection(summary string) string {
	summary = strings.TrimSpace(summary)
	for _, marker := range []string{"【进展】", "【目标】"} {
		if idx := strings.Index(summary, marker); idx >= 0 {
			summary = strings.TrimSpace(summary[:idx])
		}
	}
	return summary
}

// FormatStructuredGlobal 格式化全局画像块（注入用）
func FormatStructuredGlobal(background, goal, preference string) string {
	background = strings.TrimSpace(background)
	goal = strings.TrimSpace(goal)
	preference = strings.TrimSpace(preference)
	if background == "" && goal == "" && preference == "" {
		return ""
	}
	var parts []string
	if background != "" {
		parts = append(parts, "【背景】"+background)
	}
	if preference != "" {
		if len(parts) > 0 {
			parts[len(parts)-1] += "；讲解偏好：" + preference
		} else {
			parts = append(parts, "【背景】讲解偏好："+preference)
		}
	}
	if goal != "" {
		parts = append(parts, "【目标】"+goal)
	}
	return strings.Join(parts, "\n")
}

// ComposeLegacySummary 派生兼容 profile_summary（不含按课进展）
func ComposeLegacySummary(background, goal, preference string) string {
	out := FormatStructuredGlobal(background, goal, preference)
	if utf8.RuneCountInString(out) > maxProfileSummaryRunes {
		out = truncateRunesStorage(out, maxProfileSummaryRunes)
	}
	return out
}

// globalProfileText 优先结构化列，回退 legacy 解析
func (s *Store) globalProfileText(u *User) string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.ProfileBackground) != "" || strings.TrimSpace(u.ProfileGoal) != "" || strings.TrimSpace(u.ProfilePreference) != "" {
		return FormatStructuredGlobal(u.ProfileBackground, u.ProfileGoal, u.ProfilePreference)
	}
	return ParseBackgroundGoal(u.ProfileSummary)
}

// ComposeForBuild 建课/Planner 注入：仅全局背景与目标
func (s *Store) ComposeForBuild(userID string) (string, error) {
	u, err := s.GetUser(userID)
	if err != nil {
		return "", err
	}
	return s.globalProfileText(u), nil
}

// ComposeForCoach Coach 注入：全局 + 本课摘要
func (s *Store) ComposeForCoach(userID, domainID string) (string, error) {
	u, err := s.GetUser(userID)
	if err != nil {
		return "", err
	}
	global := s.globalProfileText(u)
	dp, err := s.GetDomainProfile(userID, domainID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if global != "" {
		b.WriteString(global)
	}
	if dp != nil && strings.TrimSpace(dp.Summary) != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("【本课进展】")
		b.WriteString(strings.TrimSpace(dp.Summary))
	}
	return strings.TrimSpace(b.String()), nil
}

// WriteGlobalProfile 写入结构化全局画像并派生 profile_summary
func (s *Store) WriteGlobalProfile(userID, background, goal, preference string) error {
	background = strings.TrimSpace(background)
	goal = strings.TrimSpace(goal)
	preference = strings.TrimSpace(preference)
	if utf8.RuneCountInString(background) > maxGlobalBackgroundRunes {
		background = truncateRunesStorage(background, maxGlobalBackgroundRunes)
	}
	if utf8.RuneCountInString(goal) > maxGlobalGoalRunes {
		goal = truncateRunesStorage(goal, maxGlobalGoalRunes)
	}
	if utf8.RuneCountInString(preference) > maxGlobalPreferenceRunes {
		preference = truncateRunesStorage(preference, maxGlobalPreferenceRunes)
	}
	legacy := ComposeLegacySummary(background, goal, preference)
	res, err := s.db.Exec(
		`UPDATE users SET profile_background = ?, profile_goal = ?, profile_preference = ?, profile_summary = ? WHERE id = ?`,
		background, goal, preference, legacy, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("角色不存在")
	}
	return nil
}

// SyncLegacySummaryFromStructured 将结构化列重算写入 profile_summary
func (s *Store) SyncLegacySummaryFromStructured(userID string) error {
	u, err := s.GetUser(userID)
	if err != nil {
		return err
	}
	legacy := ComposeLegacySummary(u.ProfileBackground, u.ProfileGoal, u.ProfilePreference)
	return s.UpdateUserProfileSummary(userID, legacy)
}
