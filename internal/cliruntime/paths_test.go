package cliruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	dir := t.TempDir()
	coach := filepath.Join(dir, "coach")
	if err := os.MkdirAll(filepath.Join(coach, "domains"), 0o755); err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePaths(coach)
	if err != nil {
		t.Fatal(err)
	}
	if paths.CoachRoot != coach {
		t.Fatalf("CoachRoot=%q", paths.CoachRoot)
	}
	if paths.DBPath != filepath.Join(coach, "data", "regulus.db") {
		t.Fatalf("DBPath=%q", paths.DBPath)
	}
	if paths.BinPath != filepath.Join(coach, "bin", "regulus") {
		t.Fatalf("BinPath=%q", paths.BinPath)
	}
}

func TestDoctorWithoutLLM(t *testing.T) {
	dir := t.TempDir()
	coach := filepath.Join(dir, "coach")
	if err := os.MkdirAll(filepath.Join(coach, "domains"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLM_API_KEY", "")

	rt, err := Open(coach)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	report := rt.Doctor(t.Context())
	if report.CoachRoot != coach {
		t.Fatalf("CoachRoot=%q", report.CoachRoot)
	}
	if report.LLMConfigured {
		t.Fatal("expected LLM not configured")
	}
	if report.DomainCount < 0 {
		t.Fatal("invalid domain count")
	}
}
