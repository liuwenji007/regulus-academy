package api

import (
	"net/http"
)

func (h *Handler) cloudSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.cloudEnabled() {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; font-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) wrapCloudMiddleware(handler http.Handler) http.Handler {
	handler = h.cloudSecurityMiddleware(handler)
	if h.cloudEnabled() && h.cloud.RateLimiter() != nil {
		handler = h.cloud.RateLimiter().Middleware(handler)
	}
	return handler
}
