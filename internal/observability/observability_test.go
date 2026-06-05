package observability

import (
	"context"
	"testing"
	"unicode/utf8"
)

func TestObserveChatCompletionDisabled(t *testing.T) {
	globalCfg.Enabled = false
	called := false
	out, err := ObserveChatCompletion(context.Background(), "deepseek", "m", nil, 0.6, false,
		func(context.Context) (string, error) {
			called = true
			return "ok", nil
		})
	if err != nil || out != "ok" || !called {
		t.Fatalf("noop path: out=%q err=%v called=%v", out, err, called)
	}
}

func TestStartChildSpanPreservesParentMeta(t *testing.T) {
	globalCfg.Enabled = false
	parent, endParent := Trace(context.Background(), TraceMeta{Name: "domain.build", UserID: "u1"})
	defer endParent()
	ctx, endChild := StartChildSpan(parent, "domain.intent", TraceMeta{Input: "Rust"})
	defer endChild()
	meta, ok := TraceMetaFromContext(ctx)
	if !ok || meta.UserID != "u1" {
		t.Fatalf("parent UserID lost: meta=%+v ok=%v", meta, ok)
	}
	if meta.Input != "Rust" {
		t.Fatalf("child input=%q", meta.Input)
	}
}

func TestTraceDisabled(t *testing.T) {
	globalCfg.Enabled = false
	ctx, end := Trace(context.Background(), TraceMeta{Name: "coach.message", UserID: "u1"})
	end()
	if _, ok := TraceMetaFromContext(ctx); !ok {
		t.Fatal("expected trace meta on context even when disabled")
	}
}

func TestOTLPEndpointUsesTracesPath(t *testing.T) {
	cfg := Config{BaseURL: "https://jp.cloud.langfuse.com"}
	want := "https://jp.cloud.langfuse.com/api/public/otel/v1/traces"
	if got := cfg.OTLPEndpoint(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTruncateUTF8Safe(t *testing.T) {
	s := "学习Rust语言编程"
	got := truncate(s, 4)
	if len(got) == 0 {
		t.Fatal("expected non-empty")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if got == s {
		t.Fatal("expected truncation")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("LANGFUSE_ENABLED", "")
	t.Setenv("LANGFUSE_LOG_CONTENT", "")
	cfg := LoadConfigFromEnv()
	if cfg.Enabled {
		t.Fatal("expected LANGFUSE_ENABLED false by default")
	}
	if !cfg.LogContent {
		t.Fatal("expected LANGFUSE_LOG_CONTENT true by default")
	}
}
