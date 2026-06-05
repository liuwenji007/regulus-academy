package domain

// minExerciseIdeasRequired 与建树 prompt / 硬校验 / 程序化质检统一：
// core 仅 1 条时至少 1 条 exercise_ideas；≥2 条时至少 2 条（不必与 core 条数相等）。
func minExerciseIdeasRequired(coreCount int) int {
	if coreCount <= 0 {
		return 0
	}
	if coreCount == 1 {
		return 1
	}
	return 2
}
