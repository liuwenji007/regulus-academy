package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHTTPTimeoutFromEnv_default(t *testing.T) {
	t.Setenv("REGULUS_LLM_TIMEOUT_SEC", "")
	if got := HTTPTimeoutFromEnv(); got != 240*time.Second {
		t.Fatalf("default=%v", got)
	}
}

func TestDomainBuildTimeoutFromEnv_custom(t *testing.T) {
	t.Setenv("REGULUS_DOMAIN_BUILD_TIMEOUT_SEC", "600")
	if got := DomainBuildTimeoutFromEnv(); got != 600*time.Second {
		t.Fatalf("custom=%v", got)
	}
}

func TestIsTimeoutErr(t *testing.T) {
	if !IsTimeoutErr(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded")
	}
	if !IsTimeoutErr(errors.New(`Post "https://x": context deadline exceeded`)) {
		t.Fatal("wrapped deadline")
	}
	if IsTimeoutErr(errors.New("other")) {
		t.Fatal("should not match")
	}
}
