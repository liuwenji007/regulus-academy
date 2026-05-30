package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateEnvKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	initial := "PORT=8080\n# comment\nGATEWAY_ENABLED=false\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	err := UpdateEnvKeys(path, map[string]string{
		"GATEWAY_ENABLED":    "true",
		"TELEGRAM_BOT_TOKEN": "abc123",
		"NEW_KEY":            "value",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "GATEWAY_ENABLED=true") {
		t.Fatalf("unexpected content:\n%s", content)
	}
	if os.Getenv("GATEWAY_ENABLED") != "true" {
		t.Fatal("expected env to be updated")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
