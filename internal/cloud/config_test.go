package cloud

import (
	"os"
	"testing"
)

func TestLoadConfigSelfHosted(t *testing.T) {
	t.Setenv("REGULUS_DEPLOYMENT", "")
	cfg := LoadConfig()
	if cfg.Enabled() {
		t.Fatal("expected cloud disabled when REGULUS_DEPLOYMENT unset")
	}
}

func TestLoadConfigCloudOK(t *testing.T) {
	t.Setenv("REGULUS_DEPLOYMENT", "cloud")
	t.Setenv("ADMIN_TOKEN", "test-admin")
	t.Setenv("REGULUS_CLOUD_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	defer func() {
		_ = os.Unsetenv("REGULUS_DEPLOYMENT")
		_ = os.Unsetenv("ADMIN_TOKEN")
		_ = os.Unsetenv("REGULUS_CLOUD_ENCRYPTION_KEY")
	}()
	cfg := LoadConfig()
	if !cfg.Enabled() {
		t.Fatal("expected cloud enabled with required env")
	}
	if cfg.QuotaDailyMessages <= 0 {
		t.Fatalf("expected positive daily quota, got %d", cfg.QuotaDailyMessages)
	}
}
