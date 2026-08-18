package api

import (
	"net/http"
	"strings"
)

func (h *Handler) getDomainSourceMaterial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	mat, err := h.store.GetDomainSourceMaterial(userID(r), id)
	if err != nil {
		if strings.Contains(err.Error(), "没有导入原文") || strings.Contains(err.Error(), "不存在") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mat)
}
