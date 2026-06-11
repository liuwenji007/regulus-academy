package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/regulus-academy/regulus-academy/internal/cloud"
	"github.com/regulus-academy/regulus-academy/internal/llm"
)

func (h *Handler) cloudEnabled() bool {
	return h.cloud != nil && h.cloud.Config().Enabled()
}

func (h *Handler) cloudUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := userID(r)
	if !h.cloudEnabled() {
		return uid, true
	}
	if err := h.cloud.ValidateUserID(uid); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return "", false
	}
	return uid, true
}

func (h *Handler) prepareCloudLLM(ctx context.Context, userID, callKind string) (context.Context, llm.Provider, string, error) {
	if !h.cloudEnabled() {
		return ctx, h.llmClient(), "platform", nil
	}
	client, billedTo, err := h.cloud.ResolveLLM(userID)
	if err != nil {
		return ctx, nil, "", err
	}
	ctx = llm.WithProvider(ctx, client)
	ctx = llm.WithUsageReporter(ctx, func(u llm.TokenUsage) {
		_ = h.cloud.RecordTokenUsage(userID, callKind, billedTo, u.PromptTokens, u.CompletionTokens, u.TotalTokens)
	})
	return ctx, client, billedTo, nil
}

func (h *Handler) writeQuotaExceeded(w http.ResponseWriter) {
	writeJSON(w, http.StatusPaymentRequired, map[string]any{
		"error":     "今日免费额度已用尽",
		"code":      "quota_exceeded",
		"needsByok": true,
	})
}

func (h *Handler) checkCoachQuota(w http.ResponseWriter, userID string) bool {
	if !h.cloudEnabled() {
		return true
	}
	if err := h.cloud.CheckMessageQuota(userID); err != nil {
		if errors.Is(err, cloud.ErrQuotaExceeded) {
			h.writeQuotaExceeded(w)
			return false
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	return true
}

func (h *Handler) recordCoachMessage(userID string) {
	if h.cloudEnabled() {
		_ = h.cloud.RecordMessageUsage(userID)
	}
}

func (h *Handler) checkBuildSlot(w http.ResponseWriter, uid string) bool {
	if !h.cloudEnabled() {
		return true
	}
	n, err := h.store.CountRunningBuildJobsForUser(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if n > 0 {
		writeError(w, http.StatusTooManyRequests, "你已有建课任务进行中，请稍候")
		return false
	}
	if h.cloud.BuildLimiter() != nil && !h.cloud.BuildLimiter().TryAcquire() {
		writeError(w, http.StatusTooManyRequests, "系统建课繁忙，请稍后再试")
		return false
	}
	return true
}
