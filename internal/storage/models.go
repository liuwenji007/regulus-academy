package storage

import "time"

// DefaultUserID MVP 单用户 ID
const DefaultUserID = "default"

// Domain 知识领域
type Domain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug,omitempty"`
	Source    string    `json:"source,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	TreeJSON  string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// DomainSummary 课程列表摘要（含进度）
type DomainSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug,omitempty"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	NodeTotal int       `json:"nodeTotal"`
	Completed int       `json:"completed"`
}

// KnowledgeTree 三层知识树结构
type KnowledgeTree struct {
	DomainID    string       `json:"domainId"`
	DomainName  string       `json:"domainName"`
	Layers      []TreeLayer  `json:"layers"`
}

// TreeLayer 知识树层级
type TreeLayer struct {
	Key   string     `json:"key"`
	Label string     `json:"label"`
	Time  string     `json:"time"`
	Goal  string     `json:"goal"`
	Nodes []TreeNode `json:"nodes"`
}

// TreeNode 知识树节点
type TreeNode struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

// UserProgress 学习进度
type UserProgress struct {
	UserID    string    `json:"userId"`
	DomainID  string    `json:"domainId"`
	NodeKey   string    `json:"nodeKey"`
	Layer     string    `json:"layer"`
	Status    string    `json:"status"`
	Mastery   float64   `json:"mastery"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Session 教学会话
type Session struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	DomainID    string    `json:"domainId"`
	DomainSlug  string    `json:"domainSlug"`
	NodeKey     string    `json:"nodeKey"`
	Status      string    `json:"status"`
	Phase       string    `json:"phase"`
	ContextJSON string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SessionContext 会话上下文（存 context_json）
type SessionContext struct {
	Exercise       *ExerciseContext `json:"exercise,omitempty"`
	ReviewedOnce   bool             `json:"reviewedOnce,omitempty"`
	DomainSlug     string           `json:"domainSlug,omitempty"`
	RecentMistakes []string         `json:"recentMistakes,omitempty"`
}

// ExerciseContext 当前练习题
type ExerciseContext struct {
	Question          string   `json:"question"`
	QuestionType      string   `json:"questionType"`
	ReinforcedConcepts []string `json:"reinforcedConcepts,omitempty"`
}

// SessionMessage 会话消息
type SessionMessage struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"sessionId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// Mistake 错题记录
type Mistake struct {
	ID                  int64      `json:"id"`
	UserID              string     `json:"userId"`
	DomainID            string     `json:"domainId"`
	NodeKey             string     `json:"nodeKey"`
	Concept             string     `json:"concept"`
	WrongCount          int        `json:"wrongCount"`
	ReinforcementCount  int        `json:"reinforcementCount"`
	LastWrong           *time.Time `json:"lastWrong,omitempty"`
}
