package agent

import "strings"

// wantsExercise 用户明确请求进入练习阶段
func wantsExercise(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"开始练习", "准备好了", "开始做题", "出题", "来一题",
		"再练一题", "再做一题", "再来一题", "再来一道",
	}
	for _, t := range triggers {
		if m == t || strings.Contains(m, t) {
			return true
		}
	}
	return false
}

// wantsBackToExplain 练习阶段请求回到讲解
func wantsBackToExplain(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{"不懂", "不明白", "回讲解", "重新讲", "再讲", "讲解", "解释一下"}
	for _, t := range triggers {
		if strings.Contains(m, t) {
			return true
		}
	}
	return false
}

// wantsNewExercise 练习阶段请求换题
func wantsNewExercise(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{"换一题", "换题", "重新出题", "再来一题", "再来一道", "另一题"}
	for _, t := range triggers {
		if strings.Contains(m, t) {
			return true
		}
	}
	return false
}

// wantsRealWorldCase 用户请求结合生产/工作场景的实际案例
func wantsRealWorldCase(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"实际案例", "生产案例", "真实场景", "真实案例", "结合实际",
		"工作场景", "生产环境", "代码怎么写", "怎么落地",
	}
	for _, t := range triggers {
		if m == t || strings.Contains(m, t) {
			return true
		}
	}
	return false
}
