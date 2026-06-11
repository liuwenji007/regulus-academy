package api

import (
	"net/http"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func userID(r *http.Request) string {
	uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if uid == "" {
		return storage.DefaultUserID
	}
	return uid
}

// cloudAnonymousAPI 无需 X-User-Id 即可访问（访客创建/浏览本地角色列表、公开元数据）
func cloudAnonymousAPI(r *http.Request) bool {
	switch r.URL.Path {
	case "/api/cloud/info", "/api/cloud/stats":
		return true
	case "/api/users":
		return r.Method == http.MethodGet || r.Method == http.MethodPost
	default:
		return false
	}
}

func (h *Handler) userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			uid := userID(r)
			if h.cloudEnabled() {
				if uid == "" || uid == storage.DefaultUserID {
					if cloudAnonymousAPI(r) {
						next.ServeHTTP(w, r)
						return
					}
					writeError(w, http.StatusUnauthorized, "需要有效的学习角色，请先创建或选择角色")
					return
				}
				if h.cloud != nil {
					h.cloud.TouchLastSeen(uid)
				}
			}
			_ = h.store.EnsureUser(uid)
		}
		next.ServeHTTP(w, r)
	})
}
