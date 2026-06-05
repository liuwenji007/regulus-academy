package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
)

type refineProfileRequest struct {
	Supplement string `json:"supplement"`
}

func (h *Handler) refineUserProfile(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		writeError(w, http.StatusBadRequest, "请先选择学习角色")
		return
	}
	var body refineProfileRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if strings.TrimSpace(body.Supplement) == "" {
		writeError(w, http.StatusBadRequest, "补充内容不能为空")
		return
	}
	if !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if _, err := h.coach.RefineUserProfile(ctx, uid, body.Supplement); err != nil {
		if strings.Contains(err.Error(), "不能为空") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	user, err := h.store.GetUser(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) updateUserProfile(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		writeError(w, http.StatusBadRequest, "请先选择学习角色")
		return
	}
	var body struct {
		ProfileSummary string `json:"profileSummary"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if err := agent.WriteUserProfile(h.store, uid, body.ProfileSummary); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.store.GetUser(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}
