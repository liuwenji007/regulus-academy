package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (h *Handler) postDomainAudit(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkUserBuildRunning(w, uid) {
		return
	}
	if !h.checkBuildQuota(w, uid, false) {
		return
	}
	if !h.acquireGlobalBuildSlot(w) {
		return
	}
	slotHandedOff := false
	defer func() {
		if !slotHandedOff {
			h.releaseGlobalBuildSlot()
		}
	}()

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
	dom, err := h.store.GetDomain(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	job, err := h.store.CreateDomainBuildJobEx(uid, tree.DomainName, "", false, storage.DomainJobKindAudit, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.recordBuildUsage(uid)
	slotHandedOff = true
	h.runGlobalBuildJobAsync(func() {
		h.runDomainAuditJob(job.ID, uid, domainID, dom)
	})
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"jobId":  job.ID,
	})
}

func (h *Handler) postDomainOptimize(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	if !h.checkUserBuildRunning(w, uid) {
		return
	}
	if !h.checkBuildQuota(w, uid, false) {
		return
	}
	if !h.acquireGlobalBuildSlot(w) {
		return
	}
	slotHandedOff := false
	defer func() {
		if !slotHandedOff {
			h.releaseGlobalBuildSlot()
		}
	}()

	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少 domain id")
		return
	}
	var body struct {
		FindingIDs []string `json:"findingIds"`
		AuditJobID string   `json:"auditJobId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if len(body.FindingIDs) == 0 {
		writeError(w, http.StatusBadRequest, "请选择至少一项可优化建议")
		return
	}

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	job, err := h.store.CreateDomainBuildJobEx(uid, tree.DomainName, "", false, storage.DomainJobKindOptimize, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.recordBuildUsage(uid)
	slotHandedOff = true
	h.runGlobalBuildJobAsync(func() {
		h.runDomainOptimizeJob(job.ID, uid, domainID, body.AuditJobID, body.FindingIDs)
	})
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"jobId":  job.ID,
	})
}

func (h *Handler) postDomainOptimizeApply(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少 domain id")
		return
	}
	var body struct {
		JobID    string   `json:"jobId"`
		PatchIDs []string `json:"patchIds"`
		Confirm  bool     `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if !body.Confirm {
		writeError(w, http.StatusBadRequest, "需要 confirm: true 才能应用优化")
		return
	}
	if strings.TrimSpace(body.JobID) == "" {
		writeError(w, http.StatusBadRequest, "缺少 jobId")
		return
	}

	job, err := h.store.GetDomainBuildJob(uid, body.JobID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if job.Status != storage.DomainBuildJobDone || job.ResultJSON == "" {
		writeError(w, http.StatusConflict, "优化任务尚未完成")
		return
	}
	if job.JobKind != storage.DomainJobKindOptimize {
		writeError(w, http.StatusBadRequest, "任务类型不是优化")
		return
	}

	var patch domain.OptimizePatch
	if err := json.Unmarshal([]byte(job.ResultJSON), &patch); err != nil {
		writeError(w, http.StatusInternalServerError, "解析优化结果失败")
		return
	}
	if patch.DomainID != "" && patch.DomainID != domainID {
		writeError(w, http.StatusBadRequest, "优化结果与课程不匹配")
		return
	}

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	nodes, err := h.registry.LoadDomainNodes(h.store, domainID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dom, derr := h.store.GetDomain(uid, domainID); derr == nil && dom.Slug != "" {
		if n2, err2 := h.registry.LoadDomainNodes(h.store, domainID, dom.Slug); err2 == nil {
			nodes = n2
		}
	}

	merged, err := domain.ApplyOptimizePatches(nodes, patch.Patches, body.PatchIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	appliedCount := len(body.PatchIDs)
	if appliedCount == 0 {
		appliedCount = len(patch.Patches)
	}
	nodesJSON, err := marshalNodesJSON(merged)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newVersion, err := h.store.UpdateDomainTreeInPlace(uid, domainID, tree, nodesJSON, nil, "course_optimize")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tree.DomainID = domainID
	writeJSON(w, http.StatusOK, map[string]any{
		"tree":        tree,
		"treeVersion": newVersion,
		"message":     fmt.Sprintf("已应用 %d 项优化，学习进度已保留", appliedCount),
	})
}

func (h *Handler) runDomainAuditJob(jobID, uid, domainID string, dom *storage.Domain) {
	ctx, cancel := context.WithTimeout(context.Background(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	reporter := &domainBuildJobReporter{store: h.store, jobID: jobID}
	reporter.ReportPhase("audit", "正在分析课程结构…")

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, err.Error())
		return
	}
	nodes, err := h.loadDomainNodesForAudit(uid, domainID, dom)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, err.Error())
		return
	}
	treeVersion, _ := h.store.GetDomainTreeVersion(domainID)

	llmClient := h.llmClient()
	if h.cloudEnabled() {
		var cerr error
		ctx, llmClient, _, cerr = h.prepareCloudLLM(ctx, uid, "domain_audit")
		if cerr != nil {
			_ = h.store.FailDomainBuildJob(jobID, cerr.Error())
			return
		}
	}

	source := dom.Source
	if source == "" {
		source = storage.DomainSourceGenerated
	}
	report, err := domain.AuditCourse(ctx, llmClient, domain.AuditCourseInput{
		DomainID:    domainID,
		DomainName:  tree.DomainName,
		Source:      source,
		TreeVersion: treeVersion,
		Tree:        tree,
		Nodes:       nodes,
	})
	if err != nil {
		msg := err.Error()
		if llm.IsTimeoutErr(err) {
			msg = "课程体检超时：请稍后重试或设置 REGULUS_COURSE_AUDIT_LLM=0 仅做规则检查"
		}
		_ = h.store.FailDomainBuildJob(jobID, msg)
		return
	}

	raw, err := json.Marshal(report)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, "序列化体检报告失败")
		return
	}
	if err := h.store.FinishDomainBuildJobEx(jobID, string(raw), "done", "体检报告已生成"); err != nil {
		log.Printf("体检任务 %s 标记完成时出错: %v", jobID, err)
	}
}

func (h *Handler) runDomainOptimizeJob(jobID, uid, domainID, auditJobID string, findingIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	reporter := &domainBuildJobReporter{store: h.store, jobID: jobID}
	reporter.ReportPhase("optimize", "正在生成优化方案…")

	var findings []domain.Finding
	if strings.TrimSpace(auditJobID) != "" {
		auditJob, err := h.store.GetDomainBuildJob(uid, auditJobID)
		if err == nil && auditJob.ResultJSON != "" {
			var report domain.CourseAuditReport
			if json.Unmarshal([]byte(auditJob.ResultJSON), &report) == nil {
				findings = domain.FindingsByIDs(report.Findings, findingIDs)
			}
		}
	}
	if len(findings) == 0 {
		// Fallback: rebuild findings from current nodes
		tree, err := h.store.GetDomainTree(uid, domainID)
		if err != nil {
			_ = h.store.FailDomainBuildJob(jobID, err.Error())
			return
		}
		dom, _ := h.store.GetDomain(uid, domainID)
		nodes, err := h.loadDomainNodesForAudit(uid, domainID, dom)
		if err != nil {
			_ = h.store.FailDomainBuildJob(jobID, err.Error())
			return
		}
		intent := domain.IntentResult{DisplayName: tree.DomainName, ScopeBreadth: domain.InferScopeFromTree(tree)}
		all := domain.FindingsByIDs(
			domain.CollectStructuredFindingsForOptimize(tree, nodes, intent),
			findingIDs,
		)
		findings = all
	}

	tree, err := h.store.GetDomainTree(uid, domainID)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, err.Error())
		return
	}
	dom, _ := h.store.GetDomain(uid, domainID)
	nodes, err := h.loadDomainNodesForAudit(uid, domainID, dom)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, err.Error())
		return
	}
	treeVersion, _ := h.store.GetDomainTreeVersion(domainID)

	llmClient := h.llmClient()
	if h.cloudEnabled() {
		var cerr error
		ctx, llmClient, _, cerr = h.prepareCloudLLM(ctx, uid, "domain_optimize")
		if cerr != nil {
			_ = h.store.FailDomainBuildJob(jobID, cerr.Error())
			return
		}
	}
	if !domain.CourseOptimizeLLMEnabled() {
		_ = h.store.FailDomainBuildJob(jobID, "未启用课程优化 LLM（REGULUS_COURSE_OPTIMIZE_LLM=0）")
		return
	}
	if !llmClient.Configured() {
		_ = h.store.FailDomainBuildJob(jobID, "未配置 LLM，无法生成优化方案")
		return
	}

	patch, err := domain.BuildOptimizePatch(ctx, llmClient, domainID, treeVersion, tree, nodes, findings)
	if err != nil {
		msg := err.Error()
		if llm.IsTimeoutErr(err) {
			msg = "课程优化超时，请减少勾选项后重试"
		}
		_ = h.store.FailDomainBuildJob(jobID, msg)
		return
	}

	raw, err := json.Marshal(patch)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, "序列化优化结果失败")
		return
	}
	if err := h.store.FinishDomainBuildJobEx(jobID, string(raw), "done", "优化方案已生成"); err != nil {
		log.Printf("优化任务 %s 标记完成时出错: %v", jobID, err)
	}
}

func (h *Handler) loadDomainNodesForAudit(uid, domainID string, dom *storage.Domain) (map[string]domain.NodeSpec, error) {
	slug := ""
	if dom != nil {
		slug = dom.Slug
	}
	return h.registry.LoadDomainNodes(h.store, domainID, slug)
}

// attachAutoAuditSummary 对 LLM 生成课做规则体检（无 LLM），结果写入 result["autoAudit"]。
// 失败时静默跳过，不阻塞建课成功返回。
func (h *Handler) attachAutoAuditSummary(uid string, result map[string]any) map[string]any {
	if result == nil {
		return result
	}
	generated, _ := result["generated"].(bool)
	if !generated {
		return result
	}
	tree, ok := result["tree"].(*storage.KnowledgeTree)
	if !ok || tree == nil || tree.DomainID == "" {
		return result
	}
	dom, err := h.store.GetDomain(uid, tree.DomainID)
	if err != nil {
		return result
	}
	nodes, err := h.loadDomainNodesForAudit(uid, tree.DomainID, dom)
	if err != nil {
		return result
	}
	treeVersion, _ := h.store.GetDomainTreeVersion(tree.DomainID)
	source := dom.Source
	if source == "" {
		source = storage.DomainSourceGenerated
	}
	// 传 nil client：仅规则检查，建课刚结束不再额外打 LLM。
	report, err := domain.AuditCourse(context.Background(), nil, domain.AuditCourseInput{
		DomainID:    tree.DomainID,
		DomainName:  tree.DomainName,
		Source:      source,
		TreeVersion: treeVersion,
		Tree:        tree,
		Nodes:       nodes,
	})
	if err != nil || report == nil {
		return result
	}
	result["autoAudit"] = map[string]any{
		"score":     report.Summary.Score,
		"grade":     report.Summary.Grade,
		"failCount": report.Summary.FailCount,
		"warnCount": report.Summary.WarnCount,
		"infoCount": report.Summary.InfoCount,
		"headline":  report.Summary.Headline,
	}
	return result
}
