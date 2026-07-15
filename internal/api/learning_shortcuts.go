package api

import (
	"net/http"
)

func (h *Handler) learningShortcuts(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		writeError(w, http.StatusBadRequest, "请先选择学习角色")
		return
	}
	if h.shortcuts == nil {
		writeError(w, http.StatusInternalServerError, "shortcuts 服务未初始化")
		return
	}
	out, err := h.shortcuts.GetLearningShortcuts(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}
