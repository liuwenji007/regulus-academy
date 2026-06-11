package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type domainBuildJobReporter struct {
	store *storage.Store
	jobID string
}

func (r *domainBuildJobReporter) ReportPhase(phase, message string) {
	_ = r.store.UpdateDomainBuildJobProgress(r.jobID, phase, message)
}

func (h *Handler) runDomainBuildJob(jobID, uid, name, goal string, force bool) {
	if h.cloudEnabled() && h.cloud.BuildLimiter() != nil {
		defer h.cloud.BuildLimiter().Release()
	}
	ctx, cancel := context.WithTimeout(context.Background(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	reporter := &domainBuildJobReporter{store: h.store, jobID: jobID}
	ctx = domain.WithBuildProgress(ctx, reporter)

	result, err := h.buildDomainForUserWithGoal(ctx, uid, name, goal, force, false)
	if err != nil {
		msg := err.Error()
		if llm.IsTimeoutErr(err) {
			msg = "知识树生成超时：模型响应较慢。请稍后重试；或增大 REGULUS_LLM_TIMEOUT_SEC / REGULUS_DOMAIN_BUILD_TIMEOUT_SEC，设置 REGULUS_TREE_CRITIQUE=0 可减少 LLM 调用次数。"
		}
		if ferr := h.store.FailDomainBuildJob(jobID, msg); ferr != nil {
			log.Printf("建课任务 %s 标记失败时出错: %v", jobID, ferr)
		}
		return
	}

	raw, err := json.Marshal(result)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, "序列化建课结果失败")
		return
	}
	if err := h.store.FinishDomainBuildJob(jobID, string(raw)); err != nil {
		log.Printf("建课任务 %s 标记完成时出错: %v", jobID, err)
	}
}

func (h *Handler) getDomainBuildJob(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobId"))
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "缺少 jobId")
		return
	}
	job, err := h.store.GetDomainBuildJob(userID(r), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	out := map[string]any{
		"status":  job.Status,
		"phase":   job.Phase,
		"message": job.Message,
		"topic":   job.Topic,
	}
	if job.Status == storage.DomainBuildJobDone && job.ResultJSON != "" {
		var result map[string]any
		if err := json.Unmarshal([]byte(job.ResultJSON), &result); err == nil {
			out["result"] = result
		}
	}
	if job.Status == storage.DomainBuildJobFailed && job.Error != "" {
		out["error"] = job.Error
	}
	writeJSON(w, http.StatusOK, out)
}
