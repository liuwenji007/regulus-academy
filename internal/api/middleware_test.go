package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminMiddlewareOptional(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "")
	called := false
	h := adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPut, "/api/llm/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("未配置 ADMIN_TOKEN 时应放行，called=%v status=%d", called, rec.Code)
	}
}

func TestAdminMiddlewareRejectsMissingToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "secret")
	h := adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPut, "/api/llm/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("缺少 Bearer token 应返回 401，status=%d", rec.Code)
	}
}

func TestAdminMiddlewareAcceptsBearerToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "secret")
	called := false
	h := adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPut, "/api/llm/config", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("正确 Bearer token 应放行，called=%v status=%d", called, rec.Code)
	}
}
