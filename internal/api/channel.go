package api

import (
	"net/http"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (h *Handler) createChannelBindCode(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" || uid == storage.DefaultUserID {
		writeError(w, http.StatusBadRequest, "请先选择或创建学习角色")
		return
	}
	code, expires, err := h.store.CreateBindCode(uid)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code":      code,
		"expiresAt": expires.Format(time.RFC3339),
		"hint":      "在 IM 中发送：绑定 " + code,
	})
}

