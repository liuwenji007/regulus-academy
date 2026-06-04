package coachstatic

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureAt 若 dest 下尚无 protocol.md，则从嵌入包解压到 dest。
func EnsureAt(dest string) (string, error) {
	if ready(dest) {
		return dest, nil
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	if err := extractAll(dest); err != nil {
		return "", err
	}
	if !ready(dest) {
		return "", fmt.Errorf("解压后仍缺少 Coach 资源（%s）", filepath.Join(dest, "protocol.md"))
	}
	return dest, nil
}

func ready(root string) bool {
	st, err := os.Stat(filepath.Join(root, "protocol.md"))
	return err == nil && !st.IsDir()
}

func extractAll(dest string) error {
	return fs.WalkDir(FS, RootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(RootDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(FS, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// ReadFile 从嵌入包读取 regulus-coach 下的相对路径。
func ReadFile(rel string) ([]byte, error) {
	return fs.ReadFile(FS, filepath.ToSlash(filepath.Join(RootDir, rel)))
}
