package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

var fieldBackfill sync.Map

func moduleLabelsOf(tree *storage.KnowledgeTree) []string {
	if tree == nil {
		return nil
	}
	var labels []string
	for _, mod := range tree.Modules {
		if label := strings.TrimSpace(mod.Label); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

// assignDomainField 给刚创建的课程归类。失败只记日志，不阻断建课。用户改过的领域不会被覆盖。
func (h *Handler) assignDomainField(ctx context.Context, uid string, dom *storage.Domain, tree *storage.KnowledgeTree, client llm.Provider) {
	if dom == nil || strings.TrimSpace(dom.ID) == "" {
		return
	}
	if current, err := h.store.GetDomain(uid, dom.ID); err == nil {
		if current.FieldSource == "user" || strings.TrimSpace(current.Field) != "" {
			return
		}
	}
	existing, err := h.store.DistinctDomainFields(uid)
	if err != nil {
		log.Printf("读取已有领域失败 %s: %v", dom.ID, err)
		return
	}
	class, err := domain.ClassifyDomainField(ctx, client, dom.Name, dom.Slug, moduleLabelsOf(tree), existing)
	if err != nil || class == nil || strings.TrimSpace(class.Field) == "" {
		log.Printf("领域归类失败 %s: %v", dom.ID, err)
		return
	}
	if err := h.store.UpdateDomainField(uid, dom.ID, class.Field, "model"); err != nil {
		log.Printf("写入领域失败 %s: %v", dom.ID, err)
	}
}

func (h *Handler) maybeBackfillFields(uid string) {
	pending, err := h.store.ListDomainsNeedingField(uid)
	if err != nil || len(pending) == 0 {
		return
	}
	if !h.cloudEnabled() && !h.llmClient().Configured() {
		return
	}
	if _, loaded := fieldBackfill.LoadOrStore(uid, struct{}{}); loaded {
		return
	}
	go func() {
		defer fieldBackfill.Delete(uid)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		client := h.llmClient()
		if h.cloudEnabled() {
			var err error
			ctx, client, _, err = h.prepareCloudLLM(ctx, uid, "domain_field")
			if err != nil {
				log.Printf("领域补判跳过 %s: %v", uid, err)
				return
			}
		}
		for _, item := range pending {
			tree, err := h.store.GetDomainTree(uid, item.ID)
			if err != nil {
				log.Printf("领域补判读课程失败 %s: %v", item.ID, err)
				continue
			}
			h.assignDomainField(ctx, uid, &storage.Domain{ID: item.ID, Name: item.Name, Slug: item.Slug}, tree, client)
		}
	}()
}

func (h *Handler) listDomainFields(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	used, err := h.store.DistinctDomainFields(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"presets": domain.PresetFields,
		"fields":  domain.AllowedFields(used),
	})
}

func (h *Handler) patchDomainField(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	var body struct {
		Field string `json:"field"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式不正确")
		return
	}
	field := strings.TrimSpace(body.Field)
	if field == "" || utf8.RuneCountInString(field) > 12 {
		writeError(w, http.StatusBadRequest, "领域名需要 1～12 个字")
		return
	}
	uid := userID(r)
	if err := h.store.UpdateDomainField(uid, id, field, "user"); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"field": field, "fieldSource": "user"})
}
