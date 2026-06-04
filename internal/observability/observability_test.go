package observability

import (
	"context"
	"testing"
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

func TestTraceDisabled(t *testing.T) {
	globalCfg.Enabled = false
	ctx, end := Trace(context.Background(), TraceMeta{Name: "coach.message", UserID: "u1"})
	end()
	if _, ok := TraceMetaFromContext(ctx); !ok {
		t.Fatal("expected trace meta on context even when disabled")
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
