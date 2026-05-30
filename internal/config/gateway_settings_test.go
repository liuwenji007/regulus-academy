package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyGatewaySettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	SetGatewayEnvFile(path)
	t.Cleanup(func() { SetGatewayEnvFile(DefaultEnvPath) })

	payload := GatewaySettingsPayload{
		Enabled:         true,
		PublicURL:       "https://example.com",
		TelegramEnabled: true,
		TelegramBotToken: "token-abc",
		FeishuEnabled:   true,
		FeishuMode:      "websocket",
		FeishuAppID:     "cli_xxx",
		FeishuAppSecret: "secret-yyy",
	}

	if err := ApplyGatewaySettings(payload); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"GATEWAY_ENABLED=true",
		"GATEWAY_PUBLIC_URL=https://example.com",
		"TELEGRAM_BOT_TOKEN=token-abc",
		"FEISHU_APP_ID=cli_xxx",
		"FEISHU_APP_SECRET=secret-yyy",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}

	// 留空密钥应保留原值
	payload2 := GatewaySettingsPayload{
		Enabled:         true,
		TelegramEnabled: true,
		FeishuEnabled:   true,
		FeishuMode:      "websocket",
		FeishuAppID:     "cli_xxx",
	}
	if err := ApplyGatewaySettings(payload2); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(path)
	if !strings.Contains(string(data2), "TELEGRAM_BOT_TOKEN=token-abc") {
		t.Fatal("empty secret should preserve existing token")
	}
}
