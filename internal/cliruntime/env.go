package cliruntime

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv 从目录加载 .env（不覆盖已设置的环境变量）。
func LoadDotEnv(dirs ...string) {
	for _, dir := range dirs {
		path := strings.TrimSpace(dir)
		if path == "" {
			continue
		}
		loadEnvFile(filepath.Join(path, ".env"))
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
