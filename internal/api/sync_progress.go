package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type syncProgressItem struct {
	Slug      string  `json:"slug"`
	NodeKey   string  `json:"nodeKey"`
	Layer     string  `json:"layer"`
	Status    string  `json:"status"`
	Mastery   float64 `json:"mastery"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// syncProgress POST /api/sync/progress — CLI 按 slug 推送进度合并。
func (h *Handler) syncProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}
	uid := userID(r)
	var req struct {
		Items []syncProgressItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	merged := 0
	for _, item := range req.Items {
		slug := strings.TrimSpace(item.Slug)
		nodeKey := strings.TrimSpace(item.NodeKey)
		if slug == "" || nodeKey == "" {
			continue
		}
		_, tree, err := h.store.GetDomainBySlug(uid, slug)
		if err != nil || tree == nil {
			continue
		}
		existing, _ := h.store.ListProgress(uid, tree.DomainID)
		var local *storage.UserProgress
		for i := range existing {
			if existing[i].NodeKey == nodeKey {
				local = &existing[i]
				break
			}
		}
		incomingAt := time.Now().UTC()
		if item.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, item.UpdatedAt); err == nil {
				incomingAt = t.UTC()
			}
		}
		if local != nil && !local.UpdatedAt.Before(incomingAt) {
			continue
		}
		layer := item.Layer
		if layer == "" {
			layer = "entry"
		}
		status := item.Status
		if status == "" {
			status = "in_progress"
		}
		if err := h.store.UpsertProgress(storage.UserProgress{
			UserID:   uid,
			DomainID: tree.DomainID,
			NodeKey:  nodeKey,
			Layer:    layer,
			Status:   status,
			Mastery:  item.Mastery,
		}); err != nil {
			continue
		}
		merged++
	}
	writeJSON(w, http.StatusOK, map[string]any{"merged": merged})
}
