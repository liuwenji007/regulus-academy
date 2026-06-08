package agent

// ExerciseOutput 出题 JSON
type ExerciseOutput struct {
	Question           string   `json:"question"`
	QuestionType       string   `json:"question_type"`
	AnswerFormat       string   `json:"answer_format"`
	Choices            []string `json:"choices"`
	ChoiceMode         string   `json:"choice_mode"`
	CorrectChoice      string   `json:"correct_choice"`
	CorrectChoices     []string `json:"correct_choices"`
	ReinforcedConcepts []string `json:"reinforced_concepts"`
}

// ChoiceGradeVerdict 选择题程序判分结果（仅服务端使用，不下发前端）
type ChoiceGradeVerdict struct {
	Passed         bool
	UserLetters    []rune
	CorrectLetters []rune
}

// ExerciseMeta 返回给前端的当前题作答方式（不含题目正文）
type ExerciseMeta struct {
	AnswerFormat string   `json:"answerFormat"`
	Choices      []string `json:"choices,omitempty"`
	ChoiceMode   string   `json:"choiceMode,omitempty"`
}

// GradeOutput 批改 JSON
type GradeOutput struct {
	Passed          bool     `json:"passed"`
	Feedback        string   `json:"feedback"`
	MistakeConcepts []string `json:"mistake_concepts"`
	WeakPoints      []string `json:"weak_points"`
}

// MasteryCheckOutput 用户申请「已掌握/下一节」时的评估 JSON
type MasteryCheckOutput struct {
	Ready       bool     `json:"ready"`
	Feedback    string   `json:"feedback"`
	GapConcepts []string `json:"gap_concepts"`
}

// MessageResult API 返回
type MessageResult struct {
	Role            string        `json:"role"`
	Content         string        `json:"content"`
	Phase           string        `json:"phase"`
	Exercise        *ExerciseMeta `json:"exercise,omitempty"`
	NodeCompleted   bool          `json:"nodeCompleted,omitempty"`
	ProgressUpdated bool          `json:"progressUpdated,omitempty"`
	NextSessionID   string        `json:"nextSessionId,omitempty"`
	NextNodeKey     string        `json:"nextNodeKey,omitempty"`
	NextNodeTitle   string        `json:"nextNodeTitle,omitempty"`
}
