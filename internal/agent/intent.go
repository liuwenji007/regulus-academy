package agent

import "strings"

// wantsExercise 用户明确请求进入练习阶段
func wantsExercise(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"开始练习", "准备好了", "开始做题", "出题", "来一题",
		"再练一题", "再做一题", "再来一题",
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
	triggers := []string{"换一题", "换题", "重新出题", "再来一题", "另一题"}
	for _, t := range triggers {
		if strings.Contains(m, t) {
			return true
		}
	}
	return false
}
