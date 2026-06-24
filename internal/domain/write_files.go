package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteDomainFiles 将 ExportPackage 文件写入 outputDir/{slug}/ 目录。
func WriteDomainFiles(outputDir, slug string, files map[string]string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" || slug == "." || slug == ".." {
		return "", fmt.Errorf("无效 slug")
	}
	root := filepath.Join(outputDir, slug)
	for rel, content := range files {
		rel = filepath.Clean(rel)
		if rel == "." || rel == ".." || filepath.IsAbs(rel) {
			return "", fmt.Errorf("无效相对路径: %s", rel)
		}
		dest := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return "", fmt.Errorf("写入 %s 失败: %w", rel, err)
		}
	}
	return root, nil
}
