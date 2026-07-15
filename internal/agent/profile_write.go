package agent

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// WriteUserProfile 写入用户画像；优先走结构化字段，剥离【进展】散文。
// 不改 preference：遗留 ProfileSummary 路径不携带该字段，需保留已有值。
func WriteUserProfile(store *storage.Store, userID, summary string) error {
	if store == nil {
		return nil
	}
	preference := ""
	if u, err := store.GetUser(userID); err == nil && u != nil {
		preference = u.ProfilePreference
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return store.WriteGlobalProfile(userID, "", "", preference)
	}
	background := storage.ParseBackgroundGoal(summary)
	goal := extractGoalSection(summary)
	if utf8.RuneCountInString(background) > 200 {
		background = truncateRunes(background, 200)
	}
	if utf8.RuneCountInString(goal) > 150 {
		goal = truncateRunes(goal, 150)
	}
	return store.WriteGlobalProfile(userID, stripSectionMarkers(background), goal, preference)
}

func extractGoalSection(summary string) string {
	for _, marker := range []string{"【目标】", "【进展】"} {
		if idx := strings.Index(summary, marker); idx >= 0 {
			return strings.TrimSpace(summary[idx+len(marker):])
		}
	}
	return ""
}

func stripSectionMarkers(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "【背景】")
	return strings.TrimSpace(s)
}

// MigrateUserProfile 保守拆分旧画像：仅有进度的域才尝试归因。
func MigrateUserProfile(store *storage.Store, userID string) (*storage.User, error) {
	if store == nil {
		return nil, fmt.Errorf("store is nil")
	}
	user, err := store.GetUser(userID)
	if err != nil {
		return nil, err
	}
	legacy := strings.TrimSpace(user.ProfileSummary)
	if legacy == "" && user.ProfileBackground == "" {
		return user, nil
	}

	background := storage.ParseBackgroundGoal(legacy)
	if user.ProfileBackground != "" {
		background = user.ProfileBackground
	}
	progress := extractGoalSection(legacy)
	if user.ProfileGoal != "" {
		progress = user.ProfileGoal
	}

	domainIDs, err := store.ListDomainIDsWithProgress(userID)
	if err != nil {
		return nil, err
	}
	remaining := progress
	for _, domainID := range domainIDs {
		if dp, _ := store.GetDomainProfile(userID, domainID); dp != nil && strings.TrimSpace(dp.Summary) != "" {
			continue
		}
		dom, err := store.GetDomain(userID, domainID)
		if err != nil || dom == nil {
			continue
		}
		name := strings.TrimSpace(dom.Name)
		if name == "" || !strings.Contains(remaining, name) {
			continue
		}
		chunk := extractSentenceAround(remaining, name)
		if chunk == "" {
			continue
		}
		_ = store.UpsertDomainProfile(userID, domainID, chunk, time.Now().UTC())
		remaining = strings.Replace(remaining, chunk, "", 1)
	}

	goal := strings.TrimSpace(remaining)
	if err := store.WriteGlobalProfile(userID, stripSectionMarkers(background), goal, user.ProfilePreference); err != nil {
		return nil, err
	}
	return store.GetUser(userID)
}

func extractSentenceAround(text, keyword string) string {
	text = strings.TrimSpace(text)
	if text == "" || keyword == "" {
		return ""
	}
	runes := []rune(text)
	kw := []rune(keyword)
	idx := -1
	for i := 0; i+len(kw) <= len(runes); i++ {
		match := true
		for j := 0; j < len(kw); j++ {
			if runes[i+j] != kw[j] {
				match = false
				break
			}
		}
		if match {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	start := idx
	for start > 0 && runes[start-1] != '。' && runes[start-1] != '\n' {
		start--
	}
	end := idx + len(kw)
	for end < len(runes) && runes[end] != '。' && runes[end] != '\n' {
		end++
	}
	if end < len(runes) {
		end++
	}
	return strings.TrimSpace(string(runes[start:end]))
}
