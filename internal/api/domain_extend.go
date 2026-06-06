package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (h *Handler) getExtendEligibility(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少 domain id")
		return
	}

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	progress, err := h.store.ListProgress(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	minRatio := domain.ExtendMinRatioFromEnv()
	eligible, completed, total, reason := domain.ExtendEligibility(tree, progress, minRatio)
	writeJSON(w, http.StatusOK, map[string]any{
		"eligible":   eligible,
		"completed":  completed,
		"total":      total,
		"minRatio":   minRatio,
		"reason":     reason,
		"treeVersion": mustTreeVersion(h.store, domainID),
	})
}

func (h *Handler) postExtendDomain(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少 domain id")
		return
	}

	var body struct {
		Confirm bool   `json:"confirm"`
		Goal    string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if !body.Confirm {
		writeError(w, http.StatusBadRequest, "需要 confirm: true 才能扩展课程")
		return
	}

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	progress, err := h.store.ListProgress(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	minRatio := domain.ExtendMinRatioFromEnv()
	eligible, completed, total, reason := domain.ExtendEligibility(tree, progress, minRatio)
	if !eligible {
		writeError(w, http.StatusConflict, fmt.Sprintf("暂不可扩展（%d/%d，需 ≥%.0f%%）：%s", completed, total, minRatio*100, reason))
		return
	}

	dom, err := h.store.GetDomain(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	nodes, err := h.registry.LoadDomainNodes(h.store, domainID, dom.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var completedKeys []string
	for _, p := range progress {
		if p.Status == "completed" {
			completedKeys = append(completedKeys, p.NodeKey)
		}
	}

	intent := domain.IntentResult{
		Slug:         dom.Slug,
		DisplayName:  tree.DomainName,
		ScopeBreadth: domain.ScopeModerate,
		Source:       domain.SourceGenerated,
	}
	if intent.Slug == "" {
		intent.Slug = domain.Slugify(tree.DomainName)
	}

	ctx, cancel := context.WithTimeout(r.Context(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	builder := domain.NewTreeBuilder(h.registry)
	result, err := builder.Extend(ctx, h.llmClient(), intent, tree, nodes, h.userProfileSummary(uid), completedKeys, strings.TrimSpace(body.Goal))
	if err != nil {
		if llm.IsTimeoutErr(err) {
			writeError(w, http.StatusGatewayTimeout, "纵深扩展超时，请稍后重试")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	nodesJSON, err := marshalNodesJSON(result.Nodes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newVersion, err := h.store.UpdateDomainTreeInPlace(uid, domainID, result.Tree, nodesJSON, result.AddedNodeKeys, strings.TrimSpace(body.Goal))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result.Tree.DomainID = domainID
	msg := fmt.Sprintf("已追加 %d 个进阶节点，原有学习进度已保留", len(result.AddedNodeKeys))
	writeJSON(w, http.StatusOK, map[string]any{
		"tree":          result.Tree,
		"addedNodeKeys": result.AddedNodeKeys,
		"treeVersion":   newVersion,
		"message":       msg,
	})
}

func mustTreeVersion(store *storage.Store, domainID string) int {
	v, err := store.GetDomainTreeVersion(domainID)
	if err != nil {
		return 1
	}
	return v
}
