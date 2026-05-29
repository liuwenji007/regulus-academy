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
			_ = h.store.EnsureUser(userID(r))
		}
		next.ServeHTTP(w, r)
	})
}
