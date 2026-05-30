package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const DefaultEnvPath = ".env"

// UpdateEnvKeys 更新 .env 中的键值，保留其余行与注释
func UpdateEnvKeys(path string, updates map[string]string) error {
	if path == "" {
		path = DefaultEnvPath
	}
	lines, err := readEnvLines(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if lines == nil {
		lines = []string{"# Regulus Academy 配置"}
	}

	seen := make(map[string]bool)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if val, ok := updates[key]; ok {
			lines[i] = formatEnvLine(key, val)
			seen[key] = true
		}
	}

	var appended []string
	for key, val := range updates {
		if seen[key] {
			continue
		}
		appended = append(appended, formatEnvLine(key, val))
	}
	if len(appended) > 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, appended...)
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存配置失败: %w", err)
	}
	for key, val := range updates {
		_ = os.Setenv(key, val)
	}
	return nil
}

func readEnvLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func formatEnvLine(key, val string) string {
	if val == "" {
		return key + "="
	}
	if needsQuotes(val) {
		return key + "=" + `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
	}
	return key + "=" + val
}

func needsQuotes(val string) bool {
	return strings.ContainsAny(val, " #\"\t")
}
