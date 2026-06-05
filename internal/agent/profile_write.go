package agent

import (
	"strings"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// WriteUserProfile 统一写入用户画像（trim + ≤500 字截断 + 落库）。
func WriteUserProfile(store *storage.Store, userID, summary string) error {
	if store == nil {
		return nil
	}
	summary = strings.TrimSpace(summary)
	if utf8.RuneCountInString(summary) > maxProfileSummaryRunes {
		summary = truncateRunes(summary, maxProfileSummaryRunes)
	}
	return store.UpdateUserProfileSummary(userID, summary)
}
