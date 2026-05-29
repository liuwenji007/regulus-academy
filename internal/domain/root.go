package domain

import (
	"os"
	"path/filepath"
	"sync"
)

const coachDirName = "regulus-coach"

var (
	coachRootOnce sync.Once
	coachRoot     string
)

// CoachRoot 返回 regulus-coach 目录绝对路径
func CoachRoot() string {
	coachRootOnce.Do(func() {
		coachRoot = findCoachRoot()
	})
	return coachRoot
}

func findCoachRoot() string {
	if p := os.Getenv("REGULUS_COACH_ROOT"); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
			candidate := filepath.Join(d, coachDirName)
			if st, err := os.Stat(candidate); err == nil && st.IsDir() {
				return candidate
			}
		}
	}
	return coachDirName
}

// ReadCoachFile 读取 regulus-coach 下相对路径文件
func ReadCoachFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(CoachRoot(), rel))
}
