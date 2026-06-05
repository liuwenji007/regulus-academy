package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Handler HTTP API 处理器
type Handler struct {
	store    *storage.Store
	llm      atomic.Value // llm.Provider
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
	h := &Handler{
		store:    store,
		registry: domain.NewRegistry(),
		coach:    coach,
		sessions: service.NewSessionService(store, coach, llmClient),
	}
	h.llm.Store(llmClient)
	return h, nil
}

func (h *Handler) llmClient() llm.Provider {
	if v := h.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
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
	mux.HandleFunc("POST /api/llm/ping", h.llmPingProbe)
	mux.HandleFunc("GET /api/llm/info", h.llmInfo)
	mux.HandleFunc("GET /api/llm/config", h.llmConfig)
	mux.HandleFunc("PUT /api/llm/config", h.updateLLMConfig)
	mux.HandleFunc("PUT /api/llm/profiles", h.updateLLMProfiles)
	mux.HandleFunc("PUT /api/llm/active", h.activateLLMProfile)
	mux.HandleFunc("GET /api/gateway/info", h.gatewayInfo)
	mux.HandleFunc("PUT /api/gateway/config", h.updateGatewayConfig)
	mux.HandleFunc("POST /api/domain/build", h.buildDomain)
	mux.HandleFunc("GET /api/domains", h.listDomains)
	mux.HandleFunc("GET /api/domains/public", h.listPublicDomains)
	mux.HandleFunc("GET /api/domain/{id}/tree", h.getDomainTree)
	mux.HandleFunc("GET /api/domain/{id}/export", h.exportDomain)
	mux.HandleFunc("DELETE /api/domain/{id}", h.deleteDomain)
	mux.HandleFunc("POST /api/domain/{id}/regenerate", h.regenerateDomain)
	mux.HandleFunc("POST /api/session/start", h.startSession)
	mux.HandleFunc("POST /api/session/next", h.startNextSession)
	mux.HandleFunc("POST /api/session/message", h.sessionMessage)
	mux.HandleFunc("GET /api/session/{id}", h.getSession)
	mux.HandleFunc("GET /api/sessions/active", h.getActiveSession)
	mux.HandleFunc("GET /api/user/progress", h.userProgress)
	mux.HandleFunc("GET /api/users", h.listUsers)
	mux.HandleFunc("POST /api/users", h.createUser)
	mux.HandleFunc("DELETE /api/users/{id}", h.deleteUser)
	mux.HandleFunc("PATCH /api/users/profile", h.updateUserProfile)
	mux.HandleFunc("POST /api/users/{id}/onboarding", h.completeUserOnboarding)
	mux.HandleFunc("POST /api/channel/bind-code", h.createChannelBindCode)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) llmPing(w http.ResponseWriter, r *http.Request) {
	if !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := h.llmClient().Ping(ctx); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": h.llmClient().Name() + " 连接正常",
	})
}

func (h *Handler) llmInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.buildLLMConfigResponse())
}

type buildDomainRequest struct {
	Name string `json:"name"`
	// Goal 可选学习目标，用于个性化裁剪（模式 B）
	Goal string `json:"goal,omitempty"`
	// Force 为 true 时跳过与已有课程的层级冲突确认
	Force bool `json:"force,omitempty"`
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

	result, err := h.buildDomainForUserWithGoal(ctx, userID(r), name, strings.TrimSpace(req.Goal), req.Force, false)
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
	return h.buildDomainForUserWithGoal(ctx, uid, name, "", false, false)
}

func (h *Handler) buildDomainForUserWithGoal(ctx context.Context, uid, name, goal string, force bool, forceNewDomain bool) (map[string]any, error) {
	_ = force
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "domain.build", UserID: uid, Input: name,
	})
	defer endTrace()
	rawIntent, err := h.registry.ParseIntent(ctx, h.llmClient(), name)
	if err != nil {
		return nil, err
	}
	intent := h.registry.NormalizeToRootTree(rawIntent)
	profile := h.userProfileSummary(uid)

	rootSlug := intent.RootSlug
	if rootSlug == "" {
		rootSlug = intent.Slug
	}
	focusKeys := append([]string(nil), intent.FocusNodeKeys...)
	focusLabel := intent.FocusLabel

	// 已有根主题树 → 聚焦子话题并返回完整结构（重建时 forceNewDomain 跳过复用）
	if !forceNewDomain {
		if existingTree, err := h.findUserRootTree(uid, rootSlug); err == nil {
		if len(focusKeys) == 0 && intent.FocusSlug != "" {
			if keys, label, kerr := h.registry.SkillPackNodeKeys(intent.FocusSlug); kerr == nil {
				focusKeys = keys
				focusLabel = label
			}
		}
			return h.treeBuildResponse(intent, existingTree, focusKeys, focusLabel, false, "", false), nil
		}
	}

	// 兼容旧数据：仅有子话题课程（如 go-concurrency）时，聚焦该树全部节点
	if !forceNewDomain && intent.FocusSlug != "" {
		if legacyTree, legacySlug, lerr := h.findLegacySubtopicTree(uid, rootSlug); lerr == nil {
			keys := domain.CollectTreeNodeKeys(legacyTree)
			label := legacyTree.DomainName
			if len(focusKeys) == 0 {
				focusKeys = keys
			}
			if focusLabel == "" {
				focusLabel = label
			}
			intent.Reason = fmt.Sprintf("已打开「%s」；完整「%s」知识树可在输入框新建", label, domain.RootDisplayName(rootSlug))
			_ = legacySlug
			return h.treeBuildResponse(intent, legacyTree, focusKeys, focusLabel, true, intent.Reason, false), nil
		}
	}

	// 子话题 + 无根树：先建根树，并入 Skill 包节点，展示完整结构并聚焦
	if intent.FocusSlug != "" {
		if !h.llmClient().Configured() {
			return nil, fmt.Errorf("未配置 LLM，无法生成「%s」完整知识树", domain.RootDisplayName(rootSlug))
		}
		rootIntent := intent
		rootIntent.Slug = rootSlug
		rootIntent.DisplayName = domain.RootDisplayName(rootSlug)
		rootIntent.ScopeBreadth = domain.ScopeBroad
		builder := domain.NewTreeBuilder(h.registry)
		tree, nodes, err := builder.Build(ctx, h.llmClient(), rootIntent, domain.RootDisplayName(rootSlug), profile)
		if err != nil {
			return nil, err
		}
		packTree, packNodes, err := h.registry.LoadTreeAndNodes(intent.FocusSlug)
		if err != nil {
			return nil, err
		}
		mergedFocus := domain.MergeSkillPackIntoTree(tree, nodes, packTree, packNodes)
		if len(focusKeys) == 0 {
			focusKeys = mergedFocus
		}
		nodesJSON, err := marshalNodesJSON(nodes)
		if err != nil {
			return nil, err
		}
		displayName := domain.RootDisplayName(rootSlug)
		_, tree, err = h.store.CreateDomainFromTree(uid, displayName, rootSlug, tree, nodesJSON, storage.DomainSourceGenerated, forceNewDomain)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("已创建「%s」完整知识树，当前聚焦「%s」", displayName, focusLabel)
		return h.treeBuildResponse(intent, tree, focusKeys, focusLabel, true, msg, true), nil
	}

	// 独立 Skill 包（无 parent_slug）
	if intent.Source == domain.SourceSkillPack {
		return h.buildSkillPackDomain(ctx, uid, name, goal, intent, forceNewDomain)
	}

	// 宽泛主题：直接生成根树
	if !h.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法生成知识树")
	}
	builder := domain.NewTreeBuilder(h.registry)
	tree, nodes, err := builder.Build(ctx, h.llmClient(), intent, name, profile)
	if err != nil {
		return nil, err
	}
	nodesJSON, err := marshalNodesJSON(nodes)
	if err != nil {
		return nil, err
	}
	displayName := intent.DisplayName
	if displayName == "" {
		displayName = name
	}
	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, rootSlug, tree, nodesJSON, storage.DomainSourceGenerated, forceNewDomain)
	if err != nil {
		return nil, err
	}
	return h.treeBuildResponse(intent, tree, nil, "", true, "", true), nil
}

func (h *Handler) buildSkillPackDomain(ctx context.Context, uid, name, goal string, intent domain.IntentResult, forceNewDomain bool) (map[string]any, error) {
	displayName := intent.DisplayName
	if displayName == "" {
		displayName = name
	}
	profile := ""
	if user, err := h.store.GetUser(uid); err == nil && user != nil {
		profile = user.ProfileSummary
	}
	if (profile != "" || goal != "") && h.llmClient().Configured() {
		publicTree, _, err := h.registry.LoadTreeAndNodes(intent.Slug)
		if err == nil {
			ver := h.registry.LoadTreeVersion(intent.Slug)
			treeMeta, _ := h.registry.FindDomainBySlug(intent.Slug)
			sel, perErr := domain.Personalize(ctx, h.llmClient(), publicTree, treeMeta, ver, profile, goal)
			if perErr == nil {
				personalTree := domain.ApplySelection(publicTree, sel)
				selJSON, _ := domain.SelectionToJSON(sel)
				_, personalTree, err = h.store.CreatePersonalizedDomain(storage.PersonalizedDomainParams{
					UserID: uid, Name: displayName, RefSlug: intent.Slug, RefVersion: ver,
					SelectionJSON: selJSON, PersonalTree: personalTree,
				})
				if err != nil {
					return nil, err
				}
				keys := domain.CollectTreeNodeKeys(personalTree)
				return h.treeBuildResponse(intent, personalTree, keys, displayName, true, sel.Reason, false), nil
			}
		}
	}
	tree, nodes, err := h.registry.LoadTreeAndNodes(intent.Slug)
	if err != nil {
		return nil, err
	}
	nodesJSON, err := marshalNodesJSON(nodes)
	if err != nil {
		return nil, err
	}
	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, intent.Slug, tree, nodesJSON, storage.DomainSourceSkillPack, forceNewDomain)
	if err != nil {
		return nil, err
	}
	keys, label, _ := h.registry.SkillPackNodeKeys(intent.Slug)
	return h.treeBuildResponse(intent, tree, keys, label, true, "", false), nil
}

func (h *Handler) findUserRootTree(uid, rootSlug string) (*storage.KnowledgeTree, error) {
	_, tree, err := h.store.GetDomainBySlug(uid, rootSlug)
	return tree, err
}

func (h *Handler) findLegacySubtopicTree(uid, rootSlug string) (*storage.KnowledgeTree, string, error) {
	list, err := h.store.ListDomainSummaries(uid)
	if err != nil {
		return nil, "", err
	}
	for _, d := range list {
		slug := strings.ToLower(strings.TrimSpace(d.Slug))
		if slug == "" {
			continue
		}
		if domain.TopicRoot(slug) != rootSlug && !h.registry.IsSubtopicOf(slug, rootSlug) {
			continue
		}
		if slug == rootSlug {
			continue
		}
		tree, err := h.registry.ResolveTree(h.store, uid, d.ID)
		if err != nil {
			continue
		}
		return tree, slug, nil
	}
	return nil, "", fmt.Errorf("not found")
}

func marshalNodesJSON(nodes map[string]domain.NodeSpec) (string, error) {
	if len(nodes) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(nodes)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *Handler) treeBuildResponse(
	intent domain.IntentResult,
	tree *storage.KnowledgeTree,
	focusKeys []string,
	focusLabel string,
	created bool,
	message string,
	generated bool,
) map[string]any {
	out := map[string]any{
		"status":    "ready",
		"intent":    intent,
		"tree":      tree,
		"generated": generated,
	}
	if len(focusKeys) > 0 {
		out["focusNodeKeys"] = focusKeys
	}
	if focusLabel != "" {
		out["focusLabel"] = focusLabel
	}
	if message != "" {
		out["message"] = message
	}
	if !created {
		out["reused"] = true
	}
	return out
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

func (h *Handler) listPublicDomains(w http.ResponseWriter, _ *http.Request) {
	list, err := h.registry.ListPublicDomains()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.PublicDomainEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": list})
}

func (h *Handler) exportDomain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	pkg, err := h.registry.ExportDomain(h.store, userID(r), id)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (h *Handler) getDomainTree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	uid := userID(r)
	tree, err := h.registry.ResolveTree(h.store, uid, id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	domainRec, err := h.store.GetDomain(uid, id)
	if err == nil {
		if nodes, nerr := h.registry.LoadDomainNodes(h.store, id, domainRec.Slug); nerr == nil {
			domain.MergeNodeRequires(tree, nodes)
		}
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
	oldDom, err := h.store.GetDomain(userID(r), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(req.ConfirmName) != oldDom.Name {
		writeError(w, http.StatusBadRequest, "输入的课程名不匹配，请完整输入要重新生成的课程名")
		return
	}
	uid := userID(r)
	oldID := id
	name := oldDom.Name
	oldSlug := oldDom.Slug
	oldSource, _ := h.store.GetDomainSource(oldID)
	oldTree, _ := h.store.GetDomainTree(uid, oldID)

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// 释放 slug 唯一约束，以便 forceNew 插入同 slug 的新课程行。
	if err := h.store.ClearDomainSlug(uid, oldID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 先建新域再迁移进度，最后删旧域；建树/迁移失败时旧域保留（slug 已清空，可再次 regenerate）。
	result, err := h.buildDomainForRegenerate(ctx, uid, name, oldSlug, oldSource, oldID, oldTree)
	if err != nil {
		if strings.Contains(err.Error(), "未配置 LLM") {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	tree, _ := result["tree"].(*storage.KnowledgeTree)
	if tree == nil || tree.DomainID == "" {
		writeError(w, http.StatusBadGateway, "重新生成未返回有效知识树")
		return
	}
	newID := tree.DomainID
	validKeys := nodeKeySet(domain.CollectTreeNodeKeys(tree))
	migrateRes, err := h.store.MigrateProgressByNodeKey(uid, oldID, newID, validKeys, oldTree, tree)
	if err != nil {
		_ = h.store.DeleteDomain(uid, newID)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newSlug, _ := h.store.GetDomainSlug(newID)
	if _, err := h.store.MigrateSessionsByNodeKey(uid, oldID, newID, newSlug, validKeys, oldTree, tree); err != nil {
		_ = h.store.DeleteDomain(uid, newID)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("进度已迁移但会话迁移失败: %v", err))
		return
	}
	if err := h.store.DeleteDomain(uid, oldID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("进度已迁移至新课程，但删除旧课程失败（旧 %s / 新 %s）: %v", oldID, newID, err))
		return
	}
	completedBefore, _ := h.store.CountCompletedProgress(uid, oldID)
	if migrateRes.Migrated > 0 || migrateRes.Skipped > 0 || completedBefore > 0 {
		result["progressKept"] = migrateRes.Migrated
		result["progressSkipped"] = migrateRes.Skipped
		var keepMsg string
		switch {
		case migrateRes.Migrated > 0:
			keepMsg = fmt.Sprintf("已保留 %d 个已掌握节点", migrateRes.Migrated)
			if migrateRes.Skipped > 0 {
				keepMsg += fmt.Sprintf("（%d 个因新树结构变化未迁移）", migrateRes.Skipped)
			}
		case completedBefore > 0:
			keepMsg = fmt.Sprintf("原课程有 %d 个已掌握节点，新树结构变化较大未能自动迁移", completedBefore)
		}
		if keepMsg != "" {
			if prev, ok := result["message"].(string); ok && strings.TrimSpace(prev) != "" {
				result["message"] = prev + "；" + keepMsg
			} else {
				result["message"] = keepMsg
			}
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// buildDomainForRegenerate 按旧课程 slug/source 重建，并提示 LLM 复用旧 node key。
func (h *Handler) buildDomainForRegenerate(
	ctx context.Context,
	uid, name, oldSlug, oldSource, oldDomainID string,
	oldTree *storage.KnowledgeTree,
) (map[string]any, error) {
	profile := h.userProfileSummary(uid)
	preserveKeys := domain.CollectTreeNodeKeys(oldTree)

	intent, err := h.intentForRegenerate(ctx, uid, name, oldSlug, oldSource, oldDomainID)
	if err != nil {
		return nil, err
	}

	switch {
	case oldSource == storage.DomainSourceSkillPack || intent.Source == domain.SourceSkillPack:
		return h.buildSkillPackDomain(ctx, uid, name, "", intent, true)
	case oldSource == storage.DomainSourcePersonalized:
		ref, refErr := h.store.GetDomainRef(oldDomainID)
		if refErr == nil && ref != nil && ref.RefSlug != "" {
			if meta, ok := h.registry.FindDomainBySlug(ref.RefSlug); ok {
				intent = domain.IntentResult{
					Slug: ref.RefSlug, DisplayName: meta.Name, Source: domain.SourceSkillPack,
					Confidence: 1, Reason: "重建个性化课程", ScopeBreadth: domain.ScopeModerate,
				}
				if name != "" {
					intent.DisplayName = name
				}
				return h.buildSkillPackDomain(ctx, uid, name, "", intent, true)
			}
		}
	}

	if !h.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法重新生成知识树")
	}
	builder := domain.NewTreeBuilder(h.registry)
	tree, nodes, err := builder.BuildRegenerate(ctx, h.llmClient(), intent, name, profile, preserveKeys)
	if err != nil {
		return nil, err
	}
	nodesJSON, err := marshalNodesJSON(nodes)
	if err != nil {
		return nil, err
	}
	rootSlug := intent.RootSlug
	if rootSlug == "" {
		rootSlug = intent.Slug
	}
	displayName := intent.DisplayName
	if displayName == "" {
		displayName = name
	}
	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, rootSlug, tree, nodesJSON, storage.DomainSourceGenerated, true)
	if err != nil {
		return nil, err
	}
	return h.treeBuildResponse(intent, tree, nil, "", true, "", true), nil
}

func (h *Handler) intentForRegenerate(ctx context.Context, uid, name, oldSlug, oldSource, oldDomainID string) (domain.IntentResult, error) {
	_ = uid
	_ = oldDomainID
	if oldSlug != "" {
		if meta, ok := h.registry.FindDomainBySlug(oldSlug); ok {
			intent := domain.IntentResult{
				Slug: oldSlug, DisplayName: meta.Name, Source: domain.SourceSkillPack,
				Confidence: 1, Reason: "重建课程", ScopeBreadth: domain.ScopeModerate,
			}
			if name != "" {
				intent.DisplayName = name
			}
			return h.registry.NormalizeToRootTree(intent), nil
		}
		intent := domain.IntentResult{
			Slug: oldSlug, DisplayName: name, Source: domain.SourceGenerated,
			Confidence: 1, Reason: "重建课程", ScopeBreadth: domain.ScopeModerate,
		}
		if oldSource == storage.DomainSourceGenerated {
			intent.Source = domain.SourceGenerated
		}
		return h.registry.NormalizeToRootTree(intent), nil
	}
	rawIntent, err := h.registry.ParseIntent(ctx, h.llmClient(), name)
	if err != nil {
		return domain.IntentResult{}, err
	}
	return h.registry.NormalizeToRootTree(rawIntent), nil
}

func (h *Handler) userProfileSummary(uid string) string {
	if u, err := h.store.GetUser(uid); err == nil && u != nil {
		return u.ProfileSummary
	}
	return ""
}

func nodeKeySet(keys []string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			m[k] = struct{}{}
		}
	}
	return m
}

type startSessionRequest struct {
	DomainID string `json:"domainId"`
	NodeKey  string `json:"nodeKey"`
	Layer    string `json:"layer"`
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request) {
	if !h.llmClient().Configured() {
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

type startNextSessionRequest struct {
	SessionID string `json:"sessionId"`
}

func (h *Handler) startNextSession(w http.ResponseWriter, r *http.Request) {
	if !h.llmClient().Configured() {
		writeError(w, http.StatusServiceUnavailable, "未配置 LLM API Key")
		return
	}
	var req startNextSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if strings.TrimSpace(req.SessionID) == "" {
		writeError(w, http.StatusBadRequest, "sessionId 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	result, err := h.sessions.StartNextNode(ctx, userID(r), req.SessionID)
	if err != nil {
		if strings.Contains(err.Error(), "尚未完成") || strings.Contains(err.Error(), "已全部完成") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "无权") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": result.Session.ID,
		"nodeKey":   result.Session.NodeKey,
		"domainId":  result.Session.DomainID,
		"phase":     result.Session.Phase,
		"content":   result.Content,
	})
}

type sessionMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

func (h *Handler) sessionMessage(w http.ResponseWriter, r *http.Request) {
	if !h.llmClient().Configured() {
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
	payload := map[string]any{
		"sessionId": sess.ID,
		"domainId":  sess.DomainID,
		"nodeKey":   sess.NodeKey,
		"nodeTitle": nodeTitle,
		"phase":     sess.Phase,
		"messages":  msgs,
		"exercise":  sessionExerciseMeta(sess),
	}
	if sess.Phase == "completed" && tree != nil {
		if nextKey, _, nextTitle, ok := domain.NextNodeAfter(tree, sess.NodeKey); ok {
			payload["nextNodeKey"] = nextKey
			payload["nextNodeTitle"] = nextTitle
		}
	}
	writeJSON(w, http.StatusOK, payload)
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

func sessionExerciseMeta(sess *storage.Session) map[string]any {
	if sess.Phase != "exercise" {
		return nil
	}
	sctx := storage.ParseSessionContext(sess)
	if sctx.Exercise == nil {
		return nil
	}
	format := sctx.Exercise.AnswerFormat
	if format == "" {
		format = agent.NormalizeAnswerFormat("", sctx.Exercise.QuestionType)
	}
	if format == "" {
		return nil
	}
	meta := map[string]any{
		"answerFormat": format,
	}
	if len(sctx.Exercise.Choices) > 0 {
		meta["choices"] = sctx.Exercise.Choices
	}
	if sctx.Exercise.ChoiceMode != "" {
		meta["choiceMode"] = sctx.Exercise.ChoiceMode
	}
	return meta
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
