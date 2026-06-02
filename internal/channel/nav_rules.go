package channel

import (
	"regexp"
	"strings"
)

var (
	reLearnPrefix = regexp.MustCompile(`(?i)^(?:学|学习|打开|进入|看看)\s*(.+)$`)
	reNodeOrdinal = regexp.MustCompile(`第(\d+)个节点`)
)

func matchNavigationRules(text string, ctx navContext) (NavigationIntent, bool) {
	if intent, ok := matchExplicitNavigation(text); ok {
		return intent, true
	}
	if intent, ok := matchLearnOrStartNode(text, ctx); ok {
		return intent, true
	}
	return NavigationIntent{}, false
}

// matchNavigationRulesWhileLearning 节点内仅识别明确导航，避免把答疑误判为选课
func matchNavigationRulesWhileLearning(text string, ctx navContext) (NavigationIntent, bool) {
	return matchExplicitNavigation(text)
}

func matchExplicitNavigation(text string) (NavigationIntent, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return NavigationIntent{}, false
	}
	lower := strings.ToLower(text)

	if matchesNavHelp(lower) {
		return NavigationIntent{Action: NavHelp}, true
	}
	if matchesNavProgress(lower) {
		return NavigationIntent{Action: NavProgress}, true
	}
	if matchesNavCourses(lower) {
		return NavigationIntent{Action: NavListCourses}, true
	}
	if matchesNavContinue(lower) {
		return NavigationIntent{Action: NavContinue}, true
	}
	return NavigationIntent{}, false
}

func matchesNavHelp(lower string) bool {
	triggers := []string{"帮助", "help", "/help", "怎么用", "能做什么", "使用说明", "命令列表", "有哪些命令"}
	for _, t := range triggers {
		if lower == t || strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func matchesNavProgress(lower string) bool {
	triggers := []string{"进度", "完成了多少", "学完多少", "学习进度", "完成度"}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func matchesNavCourses(lower string) bool {
	triggers := []string{"我的课程", "有哪些课", "课程列表", "学了什么", "知识库", "我的课", "课表"}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func matchesNavContinue(lower string) bool {
	triggers := []string{
		"接着学", "继续学", "继续吧", "接着吧", "上次学到哪", "继续上次", "接着上次",
		"续学", "接着来", "继续来", "我回来了",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func matchLearnOrStartNode(text string, ctx navContext) (NavigationIntent, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if looksLikeQuestion(lower) {
		return NavigationIntent{}, false
	}
	if ord := extractCourseOrdinal(text); ord != "" {
		return NavigationIntent{Action: NavShowNodes, CourseRef: ord}, true
	}

	if m := reNodeOrdinal.FindStringSubmatch(text); len(m) == 2 {
		if ctx.PendingDomainID != "" || ctx.ActiveDomainID != "" || len(ctx.FlatNodes) > 0 {
			return NavigationIntent{Action: NavStartNode, NodeRef: m[1]}, true
		}
	}

	if nodeRef := extractOrdinalNodeRef(text); nodeRef != "" {
		if ctx.PendingDomainID != "" || ctx.ActiveDomainID != "" || len(ctx.FlatNodes) > 0 {
			return NavigationIntent{Action: NavStartNode, NodeRef: nodeRef}, true
		}
	}

	if domainID, remainder, ok := findCourseInText(ctx.Courses, text); ok {
		nodePart := stripNavFiller(remainder)
		courseRef := courseRefFromID(ctx.Courses, domainID)
		if nodePart != "" {
			if nodeRef := extractOrdinalNodeRef(nodePart); nodeRef != "" {
				return NavigationIntent{Action: NavStartNode, CourseRef: courseRef, NodeRef: nodeRef}, true
			}
			nodes := ctx.FlatNodes
			if ctx.PendingDomainID != domainID && ctx.ActiveDomainID != domainID {
				nodes = nil
			}
			if _, _, nok := resolveNodeRef(nodes, nodePart); nok {
				return NavigationIntent{Action: NavStartNode, CourseRef: courseRef, NodeRef: nodePart}, true
			}
		}
		if courseRef != "" {
			return NavigationIntent{Action: NavShowNodes, CourseRef: courseRef}, true
		}
	}

	if m := reLearnPrefix.FindStringSubmatch(text); len(m) == 2 {
		arg := strings.TrimSpace(m[1])
		if arg != "" {
			if domainID, _, ok := findCourseInText(ctx.Courses, arg); ok {
				arg = courseRefFromID(ctx.Courses, domainID)
			}
			if nodeRef := extractOrdinalNodeRef(arg); nodeRef != "" {
				return NavigationIntent{Action: NavStartNode, NodeRef: nodeRef}, true
			}
			return NavigationIntent{Action: NavShowNodes, CourseRef: arg}, true
		}
	}

	for _, nd := range ctx.FlatNodes {
		if nd.Title != "" && strings.Contains(lower, strings.ToLower(nd.Title)) {
			return NavigationIntent{Action: NavStartNode, NodeRef: nd.Key}, true
		}
		if nd.Key != "" && len(nd.Key) > 2 && strings.Contains(lower, strings.ToLower(nd.Key)) {
			return NavigationIntent{Action: NavStartNode, NodeRef: nd.Key}, true
		}
	}

	return NavigationIntent{}, false
}

func looksLikeQuestion(lower string) bool {
	triggers := []string{
		"什么", "为什么", "为何", "怎么", "如何", "吗", "?", "？",
		"需要哪些", "有哪些", "能不能", "是不是", "可不可以",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}
