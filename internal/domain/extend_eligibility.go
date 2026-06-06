package domain

import (
	"os"
	"strconv"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// ExtendMinRatioFromEnv 纵深扩展解锁阈值（默认 0.8）
func ExtendMinRatioFromEnv() float64 {
	raw := strings.TrimSpace(os.Getenv("REGULUS_EXTEND_MIN_RATIO"))
	if raw == "" {
		return 0.8
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || v > 1 {
		return 0.8
	}
	return v
}

// ExtendEligibility 判断用户是否可解锁纵深扩展
func ExtendEligibility(tree *storage.KnowledgeTree, progress []storage.UserProgress, minRatio float64) (eligible bool, completed, total int, reason string) {
	if tree == nil {
		return false, 0, 0, "课程不存在"
	}
	total = countTreeNodes(tree)
	if total == 0 {
		return false, 0, 0, "课程暂无节点"
	}
	completed = countCompletedProgress(progress)
	ratio := float64(completed) / float64(total)
	if minRatio <= 0 {
		minRatio = ExtendMinRatioFromEnv()
	}
	if ratio < minRatio {
		return false, completed, total, "完成度未达标"
	}
	return true, completed, total, ""
}

func countCompletedProgress(progress []storage.UserProgress) int {
	n := 0
	for _, p := range progress {
		if p.Status == "completed" {
			n++
		}
	}
	return n
}
