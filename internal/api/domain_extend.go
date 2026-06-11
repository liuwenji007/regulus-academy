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
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkBuildSlot(w, uid) {
		return
	}
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

	goal := strings.TrimSpace(body.Goal)
	job, err := h.store.CreateDomainBuildJob(uid, tree.DomainName, goal, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	go h.runDomainExtendJob(job.ID, uid, domainID, goal)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"jobId":  job.ID,
	})
}

// runDomainExtendJob 异步执行纵深扩展，结果写入 domain build job（前端轮询同一接口）
func (h *Handler) runDomainExtendJob(jobID, uid, domainID, goal string) {
	if h.cloudEnabled() && h.cloud.BuildLimiter() != nil {
		defer h.cloud.BuildLimiter().Release()
	}
	ctx, cancel := context.WithTimeout(context.Background(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	_ = h.store.UpdateDomainBuildJobProgress(jobID, "extend", "正在生成进阶节点…")
	result, err := h.extendDomainForUser(ctx, uid, domainID, goal)
	if err != nil {
		msg := err.Error()
		if llm.IsTimeoutErr(err) {
			msg = "纵深扩展超时：模型响应较慢，请稍后重试"
		}
		_ = h.store.FailDomainBuildJob(jobID, msg)
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, "序列化扩展结果失败")
		return
	}
	if err := h.store.FinishDomainBuildJob(jobID, string(raw)); err != nil {
		_ = h.store.FailDomainBuildJob(jobID, err.Error())
	}
}

func (h *Handler) extendDomainForUser(ctx context.Context, uid, domainID, goal string) (map[string]any, error) {
	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		return nil, err
	}
	progress, err := h.store.ListProgress(uid, domainID)
	if err != nil {
		return nil, err
	}
	dom, err := h.store.GetDomain(uid, domainID)
	if err != nil {
		return nil, err
	}
	nodes, err := h.registry.LoadDomainNodes(h.store, domainID, dom.Slug)
	if err != nil {
		return nil, err
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
		ScopeBreadth: domain.InferScopeFromTree(tree),
		Source:       domain.SourceGenerated,
	}
	if intent.Slug == "" {
		intent.Slug = domain.Slugify(tree.DomainName)
	}

	llmClient := h.llmClient()
	if h.cloudEnabled() {
		var err error
		ctx, llmClient, _, err = h.prepareCloudLLM(ctx, uid, "domain_extend")
		if err != nil {
			return nil, err
		}
	}
	if !llmClient.Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法扩展课程")
	}

	builder := domain.NewTreeBuilder(h.registry)
	result, err := builder.Extend(ctx, llmClient, intent, tree, nodes, h.userProfileSummary(uid), completedKeys, goal)
	if err != nil {
		return nil, err
	}

	nodesJSON, err := marshalNodesJSON(result.Nodes)
	if err != nil {
		return nil, err
	}

	newVersion, err := h.store.UpdateDomainTreeInPlace(uid, domainID, result.Tree, nodesJSON, result.AddedNodeKeys, goal)
	if err != nil {
		return nil, err
	}

	result.Tree.DomainID = domainID
	return map[string]any{
		"tree":          result.Tree,
		"addedNodeKeys": result.AddedNodeKeys,
		"treeVersion":   newVersion,
		"message":       fmt.Sprintf("已追加 %d 个进阶节点，原有学习进度已保留", len(result.AddedNodeKeys)),
	}, nil
}

func mustTreeVersion(store *storage.Store, domainID string) int {
	v, err := store.GetDomainTreeVersion(domainID)
	if err != nil {
		return 1
	}
	return v
}
