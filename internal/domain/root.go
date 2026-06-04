package domain

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/regulus-academy/regulus-academy/internal/coachstatic"
)

const coachDirName = "regulus-coach"

var (
	coachRootOnce sync.Once
	coachRoot     string
)

// CoachRoot 返回 regulus-coach 目录绝对路径（磁盘 → 嵌入包解压回退）。
func CoachRoot() string {
	coachRootOnce.Do(func() {
		coachRoot = resolveCoachRoot()
	})
	return coachRoot
}

func resolveCoachRoot() string {
	var candidates []string
	if p := os.Getenv("REGULUS_COACH_ROOT"); p != "" {
		candidates = append(candidates, p)
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
			candidates = append(candidates, filepath.Join(d, coachDirName))
		}
	}
	candidates = append(candidates, coachDirName)

	seen := map[string]struct{}{}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			abs = c
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		if coachRootReady(abs) {
			return abs
		}
	}

	dest := filepath.Join(os.TempDir(), "regulus-academy-coach")
	if root, err := coachstatic.EnsureAt(dest); err == nil {
		return root
	}

	// 最后回退：保持旧行为，便于错误信息指向预期路径
	if p := os.Getenv("REGULUS_COACH_ROOT"); p != "" {
		return p
	}
	return coachDirName
}

func coachRootReady(root string) bool {
	st, err := os.Stat(filepath.Join(root, "protocol.md"))
	return err == nil && !st.IsDir()
}

// ReadCoachFile 读取 regulus-coach 下相对路径文件（磁盘优先，嵌入包回退）。
func ReadCoachFile(rel string) ([]byte, error) {
	path := filepath.Join(CoachRoot(), rel)
	b, err := os.ReadFile(path)
	if err == nil {
		return b, nil
	}
	if b, e2 := coachstatic.ReadFile(rel); e2 == nil {
		return b, nil
	}
	return nil, err
}
