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

func (h *Handler) userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			uid := userID(r)
			if h.cloudEnabled() {
				if uid == "" || uid == storage.DefaultUserID {
					// 允许无用户访问公开 cloud 元数据
					if r.URL.Path == "/api/cloud/info" || r.URL.Path == "/api/cloud/stats" {
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
