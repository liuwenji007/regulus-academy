package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Handler HTTP API 处理器
type Handler struct {
	store    *storage.Store
	llm      llm.Provider
	registry *domain.Registry
	coach    *agent.Coach
}

// NewHandler 创建处理器
func NewHandler(store *storage.Store, llmClient llm.Provider) (*Handler, error) {
	coach, err := agent.NewCoach(store, llmClient)
	if err != nil {
		return nil, err
	}
	return &Handler{
		store:    store,
		llm:      llmClient,
		registry: domain.NewRegistry(),
		coach:    coach,
	}, nil
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/llm/ping", h.llmPing)
	mux.HandleFunc("GET /api/llm/info", h.llmInfo)
	mux.HandleFunc("POST /api/domain/build", h.buildDomain)
	mux.HandleFunc("GET /api/domain/{id}/tree", h.getDomainTree)
	mux.HandleFunc("POST /api/session/start", h.startSession)
	mux.HandleFunc("POST /api/session/message", h.sessionMessage)
	mux.HandleFunc("GET /api/session/{id}", h.getSession)
	mux.HandleFunc("GET /api/sessions/active", h.getActiveSession)
	mux.HandleFunc("GET /api/user/progress", h.userProgress)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) llmPing(w http.ResponseWriter, r *http.Request) {
	if !h.llm.Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := h.llm.Ping(ctx); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": h.llm.Name() + " 连接正常",
	})
}

func (h *Handler) llmInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":    h.llm.Name(),
		"model":       h.llm.Model(),
		"configured":  h.llm.Configured(),
		"presets":     llm.ListPresets(),
	})
}

type buildDomainRequest struct {
	Name string `json:"name"`
}

func (h *Handler) buildDomain(w http.ResponseWriter, r *http.Request) {
	var req buildDomainRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "领域名称不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	intent, err := h.registry.ParseIntent(ctx, h.llm, name)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	displayName := intent.DisplayName
	if displayName == "" {
		displayName = name
	}

	var tree *storage.KnowledgeTree
	var nodes map[string]domain.NodeSpec
	source := storage.DomainSourceSkillPack

	if intent.Source == domain.SourceSkillPack {
		tree, nodes, err = h.registry.LoadTreeAndNodes(intent.Slug)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		if !h.llm.Configured() {
			writeError(w, http.StatusServiceUnavailable, "未配置 LLM，无法生成知识树")
			return
		}
		builder := domain.NewTreeBuilder(h.registry)
		tree, nodes, err = builder.Build(ctx, h.llm, intent, name)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		source = storage.DomainSourceGenerated
	}

	nodesJSON := "{}"
	if len(nodes) > 0 {
		b, err := json.Marshal(nodes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		nodesJSON = string(b)
	}

	_, tree, err = h.store.CreateDomainFromTree(displayName, intent.Slug, tree, nodesJSON, source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ready",
		"intent":     intent,
		"tree":       tree,
		"generated":  source == storage.DomainSourceGenerated,
	})
}

func (h *Handler) getDomainTree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	tree, err := h.store.GetDomainTree(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

type startSessionRequest struct {
	DomainID string `json:"domainId"`
	NodeKey  string `json:"nodeKey"`
	Layer    string `json:"layer"`
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request) {
	if !h.llm.Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	var req startSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.DomainID == "" || req.NodeKey == "" {
		writeError(w, http.StatusBadRequest, "domainId 和 nodeKey 不能为空")
		return
	}

	layer := req.Layer
	if layer == "" {
		layer = "entry"
	}

	slug, _ := h.store.GetDomainSlug(req.DomainID)

	if existing, err := h.store.FindActiveSession(storage.DefaultUserID, req.DomainID, req.NodeKey); err == nil && existing != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"sessionId": existing.ID,
			"nodeKey":   existing.NodeKey,
			"domainId":  existing.DomainID,
			"phase":     existing.Phase,
			"resumed":   true,
		})
		return
	}

	sctx := &storage.SessionContext{DomainSlug: slug}
	sess, err := h.store.CreateSession(storage.DefaultUserID, req.DomainID, slug, req.NodeKey, "explain", sctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.store.UpsertProgress(storage.UserProgress{
		UserID:   storage.DefaultUserID,
		DomainID: req.DomainID,
		NodeKey:  req.NodeKey,
		Layer:    layer,
		Status:   "in_progress",
		Mastery:  0,
	})

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	content, err := h.coach.Begin(ctx, sess)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	_, _ = h.store.AddMessage(sess.ID, "assistant", content)

	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sess.ID,
		"nodeKey":   sess.NodeKey,
		"domainId":  sess.DomainID,
		"phase":     "explain",
		"content":   content,
	})
}

type sessionMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

func (h *Handler) sessionMessage(w http.ResponseWriter, r *http.Request) {
	if !h.llm.Configured() {
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

	userRecord, err := h.store.AddMessage(req.SessionID, "user", content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	result, err := h.coach.HandleMessage(ctx, sess, content)
	if err != nil {
		_ = h.store.DeleteMessage(userRecord.ID)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if _, err := h.store.AddMessage(req.SessionID, "assistant", result.Content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.store.GetSession(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	msgs, err := h.store.ListMessages(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tree, _ := h.store.GetDomainTree(sess.DomainID)
	nodeTitle := sess.NodeKey
	if tree != nil {
		nodeTitle = domain.NodeTitle(tree, sess.NodeKey)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sess.ID,
		"domainId":  sess.DomainID,
		"nodeKey":   sess.NodeKey,
		"nodeTitle": nodeTitle,
		"phase":     sess.Phase,
		"messages":  msgs,
	})
}

func (h *Handler) userProgress(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domainId")
	list, err := h.store.ListProgress(storage.DefaultUserID, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []storage.UserProgress{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"progress": list})
}

func (h *Handler) getActiveSession(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domainId")
	nodeKey := r.URL.Query().Get("nodeKey")
	if domainID == "" || nodeKey == "" {
		writeError(w, http.StatusBadRequest, "domainId 和 nodeKey 不能为空")
		return
	}
	sess, err := h.store.FindActiveSession(storage.DefaultUserID, domainID, nodeKey)
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
		"nodeKey":   sess.NodeKey,
		"domainId":  sess.DomainID,
	})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// NewServer 创建带中间件的 HTTP 服务
func NewServer(h *Handler, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	if static != nil {
		mux.Handle("/", static)
	}
	var handler http.Handler = mux
	handler = jsonMiddleware(handler)
	handler = corsMiddleware(handler)
	return handler
}
