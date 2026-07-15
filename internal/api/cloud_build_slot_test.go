package api

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/cloud"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRunGlobalBuildJobAsyncReleasesSlot(t *testing.T) {
	t.Setenv("LANGFUSE_ENABLED", "false")
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmClient := llm.NewClient("test-key", "https://api.deepseek.com")
	cfg := cloud.Config{
		Deployment:         cloud.DeploymentCloud,
		EncryptionKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MaxBuildJobsGlobal: 1,
	}
	cloudSvc := cloud.NewService(cfg, store, llmClient)
	h, err := NewHandler(store, llmClient, cloudSvc)
	if err != nil {
		t.Fatal(err)
	}

	lim := cloudSvc.BuildLimiter()
	if !lim.TryAcquire() {
		t.Fatal("expected initial acquire")
	}

	var ran sync.WaitGroup
	ran.Add(1)
	h.runGlobalBuildJobAsync(func() {
		defer ran.Done()
	})
	ran.Wait()

	if !lim.TryAcquire() {
		t.Fatal("build slot should be released after async job completes")
	}
	lim.Release()
}

func TestReleaseGlobalBuildSlotSelfHostedNoOp(t *testing.T) {
	t.Setenv("LANGFUSE_ENABLED", "false")
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	llmClient := llm.NewClient("test-key", "https://api.deepseek.com")
	h, err := NewHandler(store, llmClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	h.runGlobalBuildJobAsync(func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("async job did not run")
	}
}
