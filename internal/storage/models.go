package storage

import "time"

// DefaultUserID MVP 单用户 ID
const DefaultUserID = "default"

// Domain 知识领域
type Domain struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug,omitempty"`
	ParentSlug string    `json:"parentSlug,omitempty"`
	Source     string    `json:"source,omitempty"`
	UserID     string    `json:"userId,omitempty"`
	TreeJSON   string    `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
}

// DomainSummary 课程列表摘要（含进度）
type DomainSummary struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug,omitempty"`
	ParentSlug string    `json:"parentSlug,omitempty"`
	Source     string    `json:"source,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	NodeTotal  int       `json:"nodeTotal"`
	Completed  int       `json:"completed"`
}

// KnowledgeTree 三层知识树结构
type KnowledgeTree struct {
	DomainID   string       `json:"domainId"`
	DomainName string       `json:"domainName"`
	Layers     []TreeLayer  `json:"layers"`
	Modules    []TreeModule `json:"modules,omitempty"`
}

// TreeModule 图谱主题模块（与进度层 layers 正交）
type TreeModule struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Goal  string   `json:"goal,omitempty"`
	Order int      `json:"order,omitempty"`
	Nodes []string `json:"nodes"`
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
	Key      string   `json:"key"`
	Title    string   `json:"title"`
	Requires []string `json:"requires,omitempty"`
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
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// SessionContext 会话上下文（存 context_json）
type SessionContext struct {
	Exercise            *ExerciseContext `json:"exercise,omitempty"`
	LastExercise        *ExerciseContext `json:"lastExercise,omitempty"`
	ReviewedOnce        bool             `json:"reviewedOnce,omitempty"`
	DomainSlug          string           `json:"domainSlug,omitempty"`
	RecentMistakes      []string         `json:"recentMistakes,omitempty"`
	TestedConcepts      []string         `json:"testedConcepts,omitempty"`
	ExplainedConcepts   []string         `json:"explainedConcepts,omitempty"`
	ConceptFollowUps    map[string]int   `json:"conceptFollowUps,omitempty"`
	DeepenedConcepts    []string         `json:"deepenedConcepts,omitempty"`
	ApplyExercisePassed bool             `json:"applyExercisePassed,omitempty"`
	OverviewDone        bool             `json:"overviewDone,omitempty"`
	SkipMasteryWarned   bool             `json:"skipMasteryWarned,omitempty"`
	PendingSkipGaps     []string         `json:"pendingSkipGaps,omitempty"`
}

// ExerciseContext 当前练习题
type ExerciseContext struct {
	Question           string   `json:"question"`
	QuestionType       string   `json:"questionType"`
	AnswerFormat       string   `json:"answerFormat"`
	Choices            []string `json:"choices,omitempty"`
	ChoiceMode         string   `json:"choiceMode,omitempty"`
	CorrectChoice      string   `json:"correctChoice,omitempty"`
	CorrectChoices     []string `json:"correctChoices,omitempty"`
	ReinforcedConcepts []string `json:"reinforcedConcepts,omitempty"`
	ExerciseLevel      string   `json:"exerciseLevel,omitempty"`
	// WrongAttempts 本题已累计答错次数（不含格式校验失败）；首次点错因，第二次讲解并换相似题。
	WrongAttempts int `json:"wrongAttempts,omitempty"`
}

// SessionMessage 会话消息
type SessionMessage struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"sessionId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// PlanningSession 行动助手规划会话
type PlanningSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Phase     string    `json:"phase"`
	PlanJSON  string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PlanningMessage 规划会话消息
type PlanningMessage struct {
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

// AsideMessage 旁路对话消息（不写入教学主线 sessions）
type AsideMessage struct {
	ID             int64     `json:"id"`
	UserID         string    `json:"userId"`
	DomainID       string    `json:"domainId"`
	NodeKey        string    `json:"nodeKey"`
	CoachSessionID string    `json:"coachSessionId,omitempty"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	AnchorText     string    `json:"anchorText,omitempty"`
	Intent         string    `json:"intent,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// TermCard 术语卡片缓存 / 术语本条目
type TermCard struct {
	ID              int64     `json:"id"`
	UserID          string    `json:"userId"`
	DomainID        string    `json:"domainId"`
	NodeKey         string    `json:"nodeKey,omitempty"`
	NormalizedTerm  string    `json:"normalizedTerm"`
	OriginalText    string    `json:"originalText"`
	CardJSON        string    `json:"cardJson"`
	HitCount        int       `json:"hitCount"`
	LastHitAt       time.Time `json:"lastHitAt"`
	CreatedAt       time.Time `json:"createdAt"`
}

// KnowledgeGap 认知缺口账本条目
type KnowledgeGap struct {
	ID              int64      `json:"id"`
	UserID          string     `json:"userId"`
	DomainID        string     `json:"domainId"`
	NodeKey         string     `json:"nodeKey,omitempty"`
	Concept         string     `json:"concept"`
	Source          string     `json:"source"` // aside_lookup | mistake | coach_gap | explicit
	HitCount        int        `json:"hitCount"`
	Severity        float64    `json:"severity"`
	MatchedDomainID string     `json:"matchedDomainId,omitempty"`
	MatchedNodeKey  string     `json:"matchedNodeKey,omitempty"`
	Reason          string     `json:"reason,omitempty"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
	LastHitAt       time.Time  `json:"lastHitAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}
