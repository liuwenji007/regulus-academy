package api

import (
	"net/http"
	"os"
	"strings"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
		for _, o := range allowed {
			if origin == o {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimSpace(auth[7:]) != token {
			writeError(w, http.StatusUnauthorized, "需要管理员 Token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func adminRoute(fn http.HandlerFunc) http.Handler {
	return adminMiddleware(http.HandlerFunc(fn))
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		next.ServeHTTP(w, r)
	})
}
