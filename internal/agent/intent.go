package agent

import "github.com/regulus-academy/regulus-academy/internal/domain"

// wantsExercise 用户明确请求进入练习阶段
func wantsExercise(msg string) bool {
	return domain.MatchTrigger("exercise", msg)
}

// wantsBackToExplain 练习阶段请求回到讲解
func wantsBackToExplain(msg string) bool {
	return domain.MatchTrigger("back_to_explain", msg)
}

// wantsNewExercise 练习阶段请求换题
func wantsNewExercise(msg string) bool {
	return domain.MatchTrigger("new_exercise", msg)
}

// wantsRealWorldCase 用户请求结合生产/工作场景的实际案例
func wantsRealWorldCase(msg string) bool {
	return domain.MatchTrigger("real_world", msg)
}

// wantsSkipMastery 用户表示已掌握、希望跳过本节点（不含纯「下一节」类表述，见 wantsStartNext）
func wantsSkipMastery(msg string) bool {
	return domain.MatchTrigger("skip_mastery", msg)
}
