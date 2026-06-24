package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type onboardingRequest struct {
	Role       string `json:"role"`
	Background string `json:"background"`
	Goal       string `json:"goal"`
	Skip       bool   `json:"skip"`
}

func (h *Handler) completeUserOnboarding(w http.ResponseWriter, r *http.Request) {
	pathID := r.PathValue("id")
	if pathID == "" {
		writeError(w, http.StatusBadRequest, "缺少角色 ID")
		return
	}
	uid := userID(r)
	if uid == "" {
		writeError(w, http.StatusBadRequest, "请先选择学习角色")
		return
	}
	if pathID != uid {
		writeError(w, http.StatusForbidden, "只能为当前学习角色完成引导")
		return
	}

	var req onboardingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}

	user, err := h.store.GetUser(uid)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if user.OnboardedAt != nil {
		writeJSON(w, http.StatusOK, user)
		return
	}

	if req.Skip {
		if err := h.store.MarkUserOnboarded(uid); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		user, _ = h.store.GetUser(uid)
		writeJSON(w, http.StatusOK, user)
		return
	}

	if h.cloudEnabled() {
		if !h.checkCoachQuota(w, uid) {
			return
		}
	} else if !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key，无法生成学生画像；可稍后再说跳过")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if h.cloudEnabled() {
		var err error
		ctx, _, _, err = h.prepareCloudLLM(ctx, uid, "coach_message")
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}

	if _, err := h.coach.InitProfileFromOnboarding(ctx, uid, req.Role, req.Background, req.Goal); err != nil {
		if strings.Contains(err.Error(), "不能为空") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	h.recordCoachMessage(uid)
	if err := h.store.MarkUserOnboarded(uid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	user, err = h.store.GetUser(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}
