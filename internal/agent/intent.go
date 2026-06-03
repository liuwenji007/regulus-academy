package agent

import "strings"

// wantsExercise 用户明确请求进入练习阶段
func wantsExercise(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"开始练习", "准备好了", "开始做题", "出题", "来一题",
		"再练一题", "再做一题", "再来一题", "再来一道",
		"继续学习", "继续学", "进入练习", "做练习", "来道练习",
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

// wantsSkipMastery 用户表示已掌握、希望跳过进入下一节
func wantsSkipMastery(msg string) bool {
	m := strings.TrimSpace(msg)
	if m == "" {
		return false
	}
	low := strings.ToLower(m)
	negatives := []string{
		"没掌握", "未掌握", "还不", "不太", "不够", "没懂", "不懂", "不会", "不清楚", "不明白",
	}
	for _, n := range negatives {
		if strings.Contains(low, n) {
			return false
		}
	}
	triggers := []string{
		"已经掌握", "已掌握", "我都掌握了", "我掌握了", "掌握了这个", "掌握了这个节点",
		"下一节", "下一章", "下一个节点", "下一节点", "进入下一",
		"跳过这", "跳过本", "先过这", "先过了", "可以先过",
		"可以过关", "过关了", "结束这一", "结束本章", "结束本节",
	}
	for _, t := range triggers {
		if strings.Contains(low, strings.ToLower(t)) {
			return true
		}
	}
	return false
}
