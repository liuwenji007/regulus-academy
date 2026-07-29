package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
)

const (
	asideMaxAnchorRunes   = 200
	asideMaxQuestionRunes = 2000
)

func (h *Handler) asideExplain(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	var req service.ExplainRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if err := validateAsideExplain(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 缓存命中不占额度、不走 Cloud LLM；未命中才校验配额并注入 provider
	ctx := r.Context()
	if !h.cloudEnabled() {
		ctx = h.withAsideLLM(ctx)
	}
	out, err := h.asideSvc.ExplainCached(uid, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if out != nil {
		writeJSON(w, http.StatusOK, out)
		return
	}
	if !h.checkCoachQuota(w, uid) {
		return
	}
	ctx, err = h.prepareAsideLLM(ctx, uid)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	out, err = h.asideSvc.ExplainGenerate(ctx, uid, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	h.recordCoachMessage(uid)
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) asideAskStream(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkCoachQuota(w, uid) {
		return
	}
	var req service.AskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if err := validateAsideAsk(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前连接不支持流式响应")
		return
	}

	ctx, err := h.prepareAsideLLM(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var writeMu sync.Mutex
	clientGone := false
	writeSSE := func(event service.AsideStreamEvent) {
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

	_, err = h.asideSvc.AskStream(ctx, uid, req, writeSSE)
	close(done)
	if err != nil {
		msg := err.Error()
		code := "error"
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(msg, "deadline") {
			code = "timeout"
			msg = "回复超时，请稍后重试"
		}
		writeSSE(service.AsideStreamEvent{Type: "error", Code: code, Error: msg})
		return
	}
	h.recordCoachMessage(uid)
}

func (h *Handler) asideListTerms(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	domainID := strings.TrimSpace(r.URL.Query().Get("domainId"))
	list, err := h.asideSvc.ListTerms(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"terms": list})
}

func (h *Handler) asideListMessages(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	domainID := strings.TrimSpace(r.URL.Query().Get("domainId"))
	list, err := h.asideSvc.ListMessages(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": list})
}

func (h *Handler) asideListGaps(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	domainID := strings.TrimSpace(r.URL.Query().Get("domainId"))
	list, err := h.asideSvc.ListGaps(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"gaps": list})
}

func (h *Handler) asideResolveGap(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "无效的缺口 id")
		return
	}
	if err := h.asideSvc.ResolveGap(uid, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// prepareAsideLLM Cloud 走 BYOK/平台额度；自托管才允许本地 aside profile。
func (h *Handler) prepareAsideLLM(ctx context.Context, userID string) (context.Context, error) {
	if h.cloudEnabled() {
		ctx, client, _, err := h.prepareCloudLLM(ctx, userID, "aside")
		if err != nil {
			return ctx, err
		}
		if client == nil || !client.Configured() {
			return ctx, fmt.Errorf("未配置 LLM API Key")
		}
		return ctx, nil
	}
	return h.withAsideLLM(ctx), nil
}

func (h *Handler) withAsideLLM(ctx context.Context) context.Context {
	state, err := config.LoadLLMProfiles()
	if err != nil {
		return ctx
	}
	aside := config.ResolveAsideProvider(state, h.llmClient())
	if aside == nil || aside == h.llmClient() {
		return ctx
	}
	return llm.WithProvider(ctx, aside)
}

func validateAsideExplain(req *service.ExplainRequest) error {
	anchor := strings.TrimSpace(req.AnchorText)
	if anchor == "" {
		return fmt.Errorf("anchorText 不能为空")
	}
	if utf8.RuneCountInString(anchor) > asideMaxAnchorRunes {
		return fmt.Errorf("划词过长（最多 %d 字）", asideMaxAnchorRunes)
	}
	req.AnchorText = anchor
	return nil
}

func validateAsideAsk(req *service.AskRequest) error {
	q := strings.TrimSpace(req.Question)
	if q == "" {
		return fmt.Errorf("question 不能为空")
	}
	if utf8.RuneCountInString(q) > asideMaxQuestionRunes {
		return fmt.Errorf("问题过长（最多 %d 字）", asideMaxQuestionRunes)
	}
	req.Question = q
	if a := strings.TrimSpace(req.AnchorText); a != "" {
		if utf8.RuneCountInString(a) > asideMaxAnchorRunes {
			return fmt.Errorf("划词过长（最多 %d 字）", asideMaxAnchorRunes)
		}
		req.AnchorText = a
	}
	return nil
}
