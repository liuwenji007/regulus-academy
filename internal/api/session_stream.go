package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/service"
)

func (h *Handler) sessionMessageStream(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkCoachQuota(w, uid) {
		return
	}
	llmClient := h.llmClient()
	if !llmClient.Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	var req sessionMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	content := strings.TrimSpace(req.Content)
	if req.SessionID == "" || content == "" {
		writeError(w, http.StatusBadRequest, "sessionId 和 content 不能为空")
		return
	}

	sess, err := h.store.GetSession(req.SessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if sess.UserID != uid {
		writeError(w, http.StatusForbidden, "无权访问此会话")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前连接不支持流式响应")
		return
	}

	ctx := r.Context()
	if h.cloudEnabled() {
		var prepErr error
		ctx, llmClient, _, prepErr = h.prepareCloudLLM(ctx, uid, "coach_message")
		if prepErr != nil {
			writeError(w, http.StatusBadGateway, prepErr.Error())
			return
		}
		if !llmClient.Configured() {
			writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var writeMu sync.Mutex
	clientGone := false
	writeSSE := func(event service.CoachStreamEvent) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if clientGone {
			return
		}
		select {
		case <-r.Context().Done():
			clientGone = true
			return
		default:
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			clientGone = true
			return
		}
		flusher.Flush()
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-r.Context().Done():
				return
			case <-ticker.C:
				writeMu.Lock()
				if !clientGone {
					if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
						clientGone = true
					} else {
						flusher.Flush()
					}
				}
				writeMu.Unlock()
			}
		}
	}()

	out, err := h.sessions.SendCoachMessageStream(ctx, uid, req.SessionID, content, writeSSE)
	close(done)

	if err != nil {
		code := "error"
		msg := err.Error()
		if errors.Is(err, service.ErrSessionBusy) {
			code = "busy"
			msg = "正在回复上一条消息，请稍候…"
		} else if strings.Contains(err.Error(), "无权") {
			code = "forbidden"
		} else if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "deadline") {
			code = "timeout"
			msg = "回复超时，请稍后重试"
		}
		writeSSE(service.CoachStreamEvent{Type: "error", Code: code, Error: msg})
		return
	}

	_ = out
	h.recordCoachMessage(uid)
}
