package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/ingest"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type sourceBuildPayload struct {
	PDFData  []byte
	Filename string
	URL      string
	Name     string
	Goal     string
	Force    bool
}

func (h *Handler) buildDomainFromSource(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	payload, err := parseSourceBuildRequest(r)
	if err != nil {
		if strings.Contains(err.Error(), "超过") || strings.Contains(err.Error(), "过大") {
			writeError(w, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	topic := strings.TrimSpace(payload.Name)
	if topic == "" {
		topic = "导入课程"
	}
	job, err := h.store.CreateDomainBuildJob(uid, topic, payload.Goal, payload.Force)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	go h.runDomainBuildFromSourceJob(job.ID, uid, payload)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"jobId":  job.ID,
	})
}

func parseSourceBuildRequest(r *http.Request) (sourceBuildPayload, error) {
	var out sourceBuildPayload
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return out, fmt.Errorf("解析上传表单失败")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return out, fmt.Errorf("缺少 PDF 文件")
		}
		defer file.Close()
		data, err := ingest.ReadLimited(file, ingest.MaxPDFBytes())
		if err != nil {
			return out, err
		}
		out.PDFData = data
		if header != nil {
			out.Filename = header.Filename
		}
		out.Name = strings.TrimSpace(r.FormValue("name"))
		out.Goal = strings.TrimSpace(r.FormValue("goal"))
		out.Force = strings.EqualFold(r.FormValue("force"), "true") || r.FormValue("force") == "1"
		return out, nil
	}

	var body struct {
		URL   string `json:"url"`
		Name  string `json:"name"`
		Goal  string `json:"goal"`
		Force bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return out, fmt.Errorf("请求体无效")
	}
	out.URL = strings.TrimSpace(body.URL)
	if out.URL == "" {
		return out, fmt.Errorf("url 不能为空")
	}
	out.Name = strings.TrimSpace(body.Name)
	out.Goal = strings.TrimSpace(body.Goal)
	out.Force = body.Force
	return out, nil
}

func (h *Handler) runDomainBuildFromSourceJob(jobID, uid string, payload sourceBuildPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), llm.DomainBuildTimeoutFromEnv())
	defer cancel()

	reporter := &domainBuildJobReporter{store: h.store, jobID: jobID}
	ctx = domain.WithBuildProgress(ctx, reporter)

	result, err := h.buildDomainFromSourceForUser(ctx, uid, payload)
	if err != nil {
		msg := err.Error()
		if llm.IsTimeoutErr(err) {
			msg = "从材料建课超时：模型响应较慢。请稍后重试；或增大 REGULUS_LLM_TIMEOUT_SEC / REGULUS_DOMAIN_BUILD_TIMEOUT_SEC。"
		}
		if ferr := h.store.FailDomainBuildJob(jobID, msg); ferr != nil {
			log.Printf("导入建课任务 %s 标记失败时出错: %v", jobID, ferr)
		}
		return
	}

	raw, err := json.Marshal(result)
	if err != nil {
		_ = h.store.FailDomainBuildJob(jobID, "序列化建课结果失败")
		return
	}
	if err := h.store.FinishDomainBuildJob(jobID, string(raw)); err != nil {
		log.Printf("导入建课任务 %s 标记完成时出错: %v", jobID, err)
	}
}

func (h *Handler) buildDomainFromSourceForUser(ctx context.Context, uid string, payload sourceBuildPayload) (map[string]any, error) {
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "domain.build_from_source", UserID: uid,
	})
	defer endTrace()

	var source ingest.Source
	var err error
	domain.ReportBuildProgress(ctx, "ingest", "正在摄取材料…")
	switch {
	case len(payload.PDFData) > 0:
		source, err = ingest.FromPDFBytes(payload.PDFData, payload.Filename)
	case payload.URL != "":
		source, err = ingest.FromURL(ctx, payload.URL)
	default:
		return nil, fmt.Errorf("缺少 PDF 或 URL")
	}
	if err != nil {
		return nil, err
	}

	domain.ReportBuildProgress(ctx, "distill", "正在蒸馏材料大纲…")
	outline, err := domain.Distill(ctx, h.llmClient(), source.Text)
	if err != nil {
		return nil, err
	}
	refOutline := domain.FormatRefOutline(outline)

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = strings.TrimSpace(outline.Title)
	}
	if name == "" {
		name = "导入课程"
	}

	goal := strings.TrimSpace(payload.Goal)
	sourceNote := "来源：" + source.Label()
	if goal == "" {
		goal = sourceNote
	} else {
		goal = goal + "；" + sourceNote
	}

	if !h.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法从材料生成知识树")
	}

	rawIntent, err := h.registry.ParseIntent(ctx, h.llmClient(), name)
	if err != nil {
		return nil, err
	}
	intent := h.registry.NormalizeToRootTree(rawIntent)
	if outline.ScopeBreadth != "" && intent.Source == domain.SourceGenerated {
		intent.ScopeBreadth = outline.ScopeBreadth
	}
	if outline.SuggestedSlug != "" && intent.Slug == "" {
		intent.Slug = domain.Slugify(outline.SuggestedSlug)
	}

	profile := h.userProfileSummary(uid)
	builder := domain.NewTreeBuilder(h.registry)
	tree, nodes, err := builder.BuildWithRefOutline(ctx, h.llmClient(), intent, name, profile, refOutline)
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

	domain.ReportBuildProgress(ctx, "saving", "正在保存课程…")
	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, rootSlug, tree, nodesJSON, storage.DomainSourceGenerated, payload.Force)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("已从「%s」生成课程「%s」", source.Label(), displayName)
	return h.treeBuildResponse(intent, tree, nil, "", true, msg, true), nil
}
