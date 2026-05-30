package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Handler HTTP API 处理器
type Handler struct {
	store    *storage.Store
	llm      llm.Provider
	registry *domain.Registry
	coach    *agent.Coach
	sessions *service.SessionService
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
		sessions: service.NewSessionService(store, coach, llmClient),
	}, nil
}

// Coach 返回教学 Agent（供 Gateway 使用）
func (h *Handler) Coach() *agent.Coach {
	return h.coach
}

// SessionService 返回会话服务
func (h *Handler) SessionService() *service.SessionService {
	return h.sessions
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/llm/ping", h.llmPing)
	mux.HandleFunc("GET /api/llm/info", h.llmInfo)
	mux.HandleFunc("POST /api/domain/build", h.buildDomain)
	mux.HandleFunc("GET /api/domains", h.listDomains)
	mux.HandleFunc("GET /api/domain/{id}/tree", h.getDomainTree)
	mux.HandleFunc("DELETE /api/domain/{id}", h.deleteDomain)
	mux.HandleFunc("POST /api/domain/{id}/regenerate", h.regenerateDomain)
	mux.HandleFunc("POST /api/session/start", h.startSession)
	mux.HandleFunc("POST /api/session/message", h.sessionMessage)
	mux.HandleFunc("GET /api/session/{id}", h.getSession)
	mux.HandleFunc("GET /api/sessions/active", h.getActiveSession)
	mux.HandleFunc("GET /api/user/progress", h.userProgress)
	mux.HandleFunc("GET /api/users", h.listUsers)
	mux.HandleFunc("POST /api/users", h.createUser)
	mux.HandleFunc("DELETE /api/users/{id}", h.deleteUser)
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

	result, err := h.buildDomainForUser(ctx, userID(r), name)
	if err != nil {
		if strings.Contains(err.Error(), "未配置 LLM") {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) buildDomainForUser(ctx context.Context, uid, name string) (map[string]any, error) {
	intent, err := h.registry.ParseIntent(ctx, h.llm, name)
	if err != nil {
		return nil, err
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
			return nil, err
		}
	} else {
		if !h.llm.Configured() {
			return nil, fmt.Errorf("未配置 LLM，无法生成知识树")
		}
		builder := domain.NewTreeBuilder(h.registry)
		tree, nodes, err = builder.Build(ctx, h.llm, intent, name)
		if err != nil {
			return nil, err
		}
		source = storage.DomainSourceGenerated
	}

	nodesJSON := "{}"
	if len(nodes) > 0 {
		b, err := json.Marshal(nodes)
		if err != nil {
			return nil, err
		}
		nodesJSON = string(b)
	}

	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, intent.Slug, tree, nodesJSON, source)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":    "ready",
		"intent":    intent,
		"tree":      tree,
		"generated": source == storage.DomainSourceGenerated,
	}, nil
}

func (h *Handler) listDomains(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListDomainSummaries(userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []storage.DomainSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": list})
}

func (h *Handler) getDomainTree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	tree, err := h.store.GetDomainTree(userID(r), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

type domainActionRequest struct {
	ConfirmName string `json:"confirmName"`
}

func (h *Handler) deleteDomain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	var req domainActionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	domain, err := h.store.GetDomain(userID(r), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(req.ConfirmName) != domain.Name {
		writeError(w, http.StatusBadRequest, "输入的课程名不匹配，请完整输入要移除的课程名")
		return
	}
	if err := h.store.DeleteDomain(userID(r), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) regenerateDomain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	var req domainActionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	domain, err := h.store.GetDomain(userID(r), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(req.ConfirmName) != domain.Name {
		writeError(w, http.StatusBadRequest, "输入的课程名不匹配，请完整输入要重新生成的课程名")
		return
	}
	name := domain.Name
	if err := h.store.DeleteDomain(userID(r), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	result, err := h.buildDomainForUser(ctx, userID(r), name)
	if err != nil {
		if strings.Contains(err.Error(), "未配置 LLM") {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	result, err := h.sessions.StartOrResumeSession(ctx, userID(r), req.DomainID, req.NodeKey, layer)
	if err != nil {
		if strings.Contains(err.Error(), "课程不存在") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if result.Resumed {
		writeJSON(w, http.StatusOK, map[string]any{
			"sessionId": result.Session.ID,
			"nodeKey":   result.Session.NodeKey,
			"domainId":  result.Session.DomainID,
			"phase":     result.Session.Phase,
			"resumed":   true,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": result.Session.ID,
		"nodeKey":   result.Session.NodeKey,
		"domainId":  result.Session.DomainID,
		"phase":     "explain",
		"content":   result.Content,
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
	if sess.UserID != userID(r) {
		writeError(w, http.StatusForbidden, "无权访问此会话")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	out, err := h.sessions.SendCoachMessage(ctx, userID(r), req.SessionID, content)
	if err != nil {
		if strings.Contains(err.Error(), "无权") {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, out.Result)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.store.GetSession(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if sess.UserID != userID(r) {
		writeError(w, http.StatusForbidden, "无权访问此会话")
		return
	}
	msgs, err := h.store.ListMessages(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tree, _ := h.store.GetDomainTree(sess.UserID, sess.DomainID)
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
	list, err := h.store.ListProgress(userID(r), domainID)
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
	sess, err := h.store.FindLatestSession(userID(r), domainID, nodeKey)
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

func (h *Handler) listUsers(w http.ResponseWriter, _ *http.Request) {
	list, err := h.store.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []storage.User{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list})
}

type createUserRequest struct {
	DisplayName string `json:"displayName"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	user, err := h.store.CreateUser(req.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type deleteUserRequest struct {
	ConfirmName string `json:"confirmName"`
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少角色 ID")
		return
	}
	var req deleteUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	user, err := h.store.GetUser(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(req.ConfirmName) != user.DisplayName {
		writeError(w, http.StatusBadRequest, "输入的角色名不匹配，请完整输入要移除的角色名")
		return
	}
	if err := h.store.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
func NewServer(h *Handler, static http.Handler, gatewayWebhooks func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	if gatewayWebhooks != nil {
		gatewayWebhooks(mux)
	}
	if static != nil {
		mux.Handle("/", static)
	}
	var handler http.Handler = mux
	handler = h.userMiddleware(handler)
	handler = jsonMiddleware(handler)
	handler = corsMiddleware(handler)
	return handler
}
