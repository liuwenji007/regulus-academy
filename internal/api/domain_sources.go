package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/ingest"
	"github.com/regulus-academy/regulus-academy/internal/search"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func isPDFName(name string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".pdf")
}

func (h *Handler) saveIngestedSource(userID, domainID string, src ingest.Source) error {
	chunks := domain.ChunkForRetrieval(src.Text)
	if _, err := h.store.InsertDomainSource(&storage.DomainSource{
		UserID:   userID,
		DomainID: domainID,
		Kind:     src.Kind,
		Filename: src.Filename,
		URL:      src.Meta.URL,
		Text:     src.Text,
	}, chunks); err != nil {
		return err
	}
	if err := domain.BindAndGround(h.store, userID, domainID); err != nil {
		log.Printf("资料绑定失败 domain=%s: %v", domainID, err)
	}
	return h.store.DeleteTermCardsForDomain(userID, domainID)
}

func (h *Handler) listDomainSources(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	if _, err := h.store.GetDomain(uid, domainID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	list, err := h.store.ListDomainSourceMeta(uid, domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []storage.DomainSourceMeta{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": list})
}

func (h *Handler) postDomainSource(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	domainID := strings.TrimSpace(r.PathValue("id"))
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "缺少领域 ID")
		return
	}
	if _, err := h.store.GetDomain(uid, domainID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	src, err := parseAppendSource(r)
	if err != nil {
		if strings.Contains(err.Error(), "超过") || strings.Contains(err.Error(), "过大") || strings.Contains(err.Error(), "过长") {
			writeError(w, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.saveIngestedSource(uid, domainID, src); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, _ := h.store.ListDomainSourceMeta(uid, domainID)
	writeJSON(w, http.StatusOK, map[string]any{
		"sources": list,
		"message": "已挂上资料，并按节点重新绑定",
	})
}

func parseAppendSource(r *http.Request) (ingest.Source, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return ingest.Source{}, err
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return ingest.Source{}, err
		}
		defer file.Close()
		data, err := ingest.ReadLimited(file, ingest.MaxPDFBytes())
		if err != nil {
			return ingest.Source{}, err
		}
		if len(data) > ingest.MaxPDFBytes() {
			return ingest.Source{}, errString("文件超过大小上限")
		}
		name := ""
		if header != nil {
			name = header.Filename
		}
		if isPDFName(name) {
			return ingest.FromPDFBytes(data, name)
		}
		return ingest.FromPlain(name, string(data))
	}
	var body struct {
		URL  string `json:"url"`
		Text string `json:"text"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, int64(ingest.MaxPDFBytes())+1)).Decode(&body); err != nil {
		return ingest.Source{}, errString("请求体无效")
	}
	if url := strings.TrimSpace(body.URL); url != "" {
		return ingest.FromURL(r.Context(), url)
	}
	if strings.TrimSpace(body.Text) != "" {
		return ingest.FromPlain(body.Name, body.Text)
	}
	return ingest.Source{}, errString("请上传文件、填写 URL 或粘贴正文")
}

type errString string

func (e errString) Error() string { return string(e) }

func (h *Handler) searchInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.cloudUserID(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": search.Configured()})
}
