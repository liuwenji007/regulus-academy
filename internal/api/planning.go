package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/service"
)

type startPlanningRequest struct {
	ForceNew bool `json:"forceNew"`
}

func (h *Handler) startPlanning(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if uid == "" {
		writeError(w, http.StatusBadRequest, "请先选择学习角色")
		return
	}
	if !h.cloudEnabled() && !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}

	var req startPlanningRequest
	_ = decodeJSON(r, &req)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if h.cloudEnabled() {
		if !h.checkCoachQuota(w, uid) {
			return
		}
		var err error
		ctx, _, _, err = h.prepareCloudLLM(ctx, uid, "coach_message")
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}

	result, err := h.planning.StartOrResume(ctx, uid, req.ForceNew)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	payload := map[string]any{
		"sessionId": result.Session.ID,
		"phase":     result.Session.Phase,
		"resumed":   result.Resumed,
		"messages":  result.Messages,
	}
	if result.Content != "" && !result.Resumed {
		payload["content"] = result.Content
	}
	if result.Plan != nil {
		payload["plan"] = result.Plan
	}
	writeJSON(w, http.StatusOK, payload)
}

type planningMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

func (h *Handler) planningMessage(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkCoachQuota(w, uid) {
		return
	}
	if !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}

	var req planningMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	content := strings.TrimSpace(req.Content)
	if req.SessionID == "" || content == "" {
		writeError(w, http.StatusBadRequest, "sessionId 和 content 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if h.cloudEnabled() {
		var err error
		ctx, _, _, err = h.prepareCloudLLM(ctx, uid, "coach_message")
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}

	out, err := h.planning.SendPlanningMessage(ctx, uid, req.SessionID, content)
	if err != nil {
		if errors.Is(err, service.ErrSessionBusy) {
			writeError(w, http.StatusConflict, "正在回复上一条消息，请稍候…")
			return
		}
		if strings.Contains(err.Error(), "无权") {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	h.recordCoachMessage(uid)
	resp := map[string]any{
		"role":      "assistant",
		"content":   out.Reply,
		"phase":     out.Phase,
		"synthesized": out.Synthesized,
	}
	if out.Plan != nil {
		resp["plan"] = out.Plan
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) getPlanningSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	uid := userID(r)
	sess, err := h.store.GetPlanningSession(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if sess.UserID != uid {
		writeError(w, http.StatusForbidden, "无权访问此会话")
		return
	}
	msgs, err := h.store.ListPlanningMessages(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	plan, _ := agent.ParsePlanningResult(sess.PlanJSON)
	payload := map[string]any{
		"sessionId": sess.ID,
		"phase":     sess.Phase,
		"status":    sess.Status,
		"messages":  msgs,
	}
	if plan != nil {
		payload["plan"] = plan
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) getActivePlanningSession(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		writeJSON(w, http.StatusOK, map[string]any{"sessionId": nil})
		return
	}
	sess, err := h.store.FindActivePlanningSession(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]any{"sessionId": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sess.ID,
		"phase":     sess.Phase,
	})
}
