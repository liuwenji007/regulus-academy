package api

import (
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func (h *Handler) registerCloudRoutes(mux *http.ServeMux) {
	if !h.cloudEnabled() {
		return
	}
	mux.HandleFunc("GET /api/cloud/info", h.cloudInfo)
	mux.HandleFunc("GET /api/cloud/stats", h.cloudStats)
	mux.HandleFunc("GET /api/user/llm-quota", h.userLLMQuota)
	mux.HandleFunc("PUT /api/user/llm-key", h.putUserLLMKey)
	mux.Handle("GET /api/admin/stats", adminRoute(h.adminStats))
	mux.Handle("GET /api/admin/users", adminRoute(h.adminUsers))
	mux.Handle("GET /api/admin/usage", adminRoute(h.adminUsage))
	mux.Handle("POST /api/admin/users/{id}/reset-quota", adminRoute(h.adminResetQuota))
}

func (h *Handler) cloudInfo(w http.ResponseWriter, _ *http.Request) {
	cfg := h.cloud.Config()
	writeJSON(w, http.StatusOK, map[string]any{
		"deployment":    cfg.Deployment,
		"githubUrl":     cfg.GithubURL,
		"docsUrl":       cfg.DocsURL,
		"demoUrl":       cfg.DemoURL,
		"demoLabel":     "Regulus Academy 在线体验",
		"selfHostHint":  "完整功能请本地 Docker 部署",
		"quotaDaily":    cfg.QuotaDailyMessages,
	})
}

func (h *Handler) cloudStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := h.cloud.PublicStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) userLLMQuota(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	q, err := h.cloud.QuotaStatus(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

type putLLMKeyRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
}

func (h *Handler) putUserLLMKey(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.cloudUserID(w, r)
	if !ok {
		return
	}
	var req putLLMKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		writeError(w, http.StatusBadRequest, "apiKey 不能为空")
		return
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = "deepseek"
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		if p, ok := llm.GetPreset(provider); ok {
			model = p.Model
		}
	}
	if err := h.cloud.SaveUserLLMKey(uid, provider, key, strings.TrimSpace(req.BaseURL), model); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q, _ := h.cloud.QuotaStatus(uid)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "quota": q})
}

func (h *Handler) adminStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := h.cloud.AdminStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) adminUsers(w http.ResponseWriter, _ *http.Request) {
	list, err := h.store.ListAdminUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []storage.AdminUserRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list})
}

func (h *Handler) adminUsage(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.store.AdminUsageByDay(14)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"byDay": rows})
}

func (h *Handler) adminResetQuota(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少用户 id")
		return
	}
	if err := h.store.ResetUserDailyQuota(id, storage.TodayUTC()); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
