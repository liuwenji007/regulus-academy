package channel

// NavAction IM 导航意图动作（规则层与 LLM 层共用）
type NavAction string

const (
	NavListCourses NavAction = "list_courses"
	NavShowNodes   NavAction = "show_nodes"
	NavStartNode   NavAction = "start_node"
	NavContinue    NavAction = "continue"
	NavProgress    NavAction = "progress"
	NavHelp        NavAction = "help"
	NavClarify     NavAction = "clarify"
)

// NavigationIntent 导航意图（courseRef/nodeRef 为序号、slug、名称或标题片段）
type NavigationIntent struct {
	Action    NavAction
	CourseRef string
	NodeRef   string
	ReplyHint string
}
