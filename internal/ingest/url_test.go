package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFromURLExtractsArticle(t *testing.T) {
	t.Setenv("REGULUS_INGEST_ALLOW_PRIVATE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>T</title></head><body>
<article><h1>Go 并发入门</h1><p>goroutine 是 Go 的轻量线程。</p><p>channel 用于协程通信。</p></article>
<nav>skip</nav></body></html>`))
	}))
	defer srv.Close()

	src, err := FromURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if src.Kind != KindURL {
		t.Fatalf("kind=%s", src.Kind)
	}
	if !strings.Contains(src.Text, "goroutine") {
		t.Fatalf("text=%q", src.Text)
	}
}

func TestFromURLRejectsInvalidScheme(t *testing.T) {
	_, err := FromURL(context.Background(), "ftp://example.com/doc")
	if err == nil {
		t.Fatal("应拒绝非 http(s) URL")
	}
}

func TestFromURLRejectsPrivateIP(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/doc",
		"http://10.0.0.1/doc",
		"http://192.168.1.1/doc",
		"http://169.254.169.254/latest/meta-data",
	}
	for _, raw := range cases {
		_, err := FromURL(context.Background(), raw)
		if err == nil {
			t.Fatalf("应拒绝内网地址: %s", raw)
		}
	}
}
