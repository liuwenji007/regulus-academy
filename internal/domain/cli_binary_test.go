package domain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeCLIPlatform(t *testing.T) {
	got, err := NormalizeCLIPlatform("darwin_arm64")
	if err != nil || got != "darwin_arm64" {
		t.Fatalf("got %q err %v", got, err)
	}
	_, err = NormalizeCLIPlatform("invalid")
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestResolveCLIBinaryHost(t *testing.T) {
	chdirCoachRoot(t)
	root := CoachRoot()
	bin := filepath.Join(root, "bin", "regulus")
	repoBin := filepath.Join(filepath.Dir(root), "bin", "regulus")
	if _, err := os.Stat(bin); err != nil {
		if _, err2 := os.Stat(repoBin); err2 != nil {
			t.Skip("跳过：未找到 bin/regulus")
		}
	}
	host := runtime.GOOS + "_" + runtime.GOARCH
	_, content, err := ResolveCLIBinary(host)
	if err != nil {
		t.Fatalf("ResolveCLIBinary: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("binary empty")
	}
}

func TestResolveCLIBinaryMissing(t *testing.T) {
	chdirCoachRoot(t)
	_, _, err := ResolveCLIBinary("linux_riscv64")
	if err == nil {
		t.Fatal("expected error for unsupported platform")
	}
}
