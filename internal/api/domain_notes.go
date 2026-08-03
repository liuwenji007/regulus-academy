package api

import (
	"net/http"
	"sort"
)

// listDomainNotes GET /api/domain/{id}/notes
// 可选 ?nodeKey= 只返回单节点笔记。
func (h *Handler) listDomainNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	uid := userID(r)
	ok, err := h.store.DomainOwnedByUser(uid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "领域不存在")
		return
	}

	nodeKey := r.URL.Query().Get("nodeKey")
	if nodeKey != "" {
		content, err := h.store.GetNodeNote(uid, id, nodeKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "读取笔记失败: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"domainId": id,
			"notes": []map[string]string{{
				"nodeKey":   nodeKey,
				"contentMd": content,
			}},
		})
		return
	}

	notes, err := h.store.ListNodeNotes(uid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取笔记失败: "+err.Error())
		return
	}
	keys := make([]string, 0, len(notes))
	for k := range notes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, map[string]string{
			"nodeKey":   k,
			"contentMd": notes[k],
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"domainId": id,
		"notes":    items,
	})
}

// listDomainMistakes GET /api/domain/{id}/mistakes
// 可选 ?nodeKey= 只返回单节点错题概念。
func (h *Handler) listDomainMistakes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	uid := userID(r)
	ok, err := h.store.DomainOwnedByUser(uid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "领域不存在")
		return
	}

	nodeKey := r.URL.Query().Get("nodeKey")
	mistakes, err := h.store.ListMistakesByNode(uid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取错题失败: "+err.Error())
		return
	}

	if nodeKey != "" {
		concepts := mistakes[nodeKey]
		if concepts == nil {
			concepts = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"domainId": id,
			"mistakes": []map[string]any{{
				"nodeKey":  nodeKey,
				"concepts": concepts,
			}},
		})
		return
	}

	keys := make([]string, 0, len(mistakes))
	for k := range mistakes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		items = append(items, map[string]any{
			"nodeKey":  k,
			"concepts": mistakes[k],
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"domainId": id,
		"mistakes": items,
	})
}
