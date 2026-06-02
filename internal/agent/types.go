package agent

// ExerciseOutput 出题 JSON
type ExerciseOutput struct {
	Question           string   `json:"question"`
	QuestionType       string   `json:"question_type"`
	AnswerFormat       string   `json:"answer_format"`
	Choices            []string `json:"choices"`
	ChoiceMode         string   `json:"choice_mode"`
	ReinforcedConcepts []string `json:"reinforced_concepts"`
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
}

// MessageResult API 返回
type MessageResult struct {
	Role            string        `json:"role"`
	Content         string        `json:"content"`
	Phase           string        `json:"phase"`
	Exercise        *ExerciseMeta `json:"exercise,omitempty"`
	NodeCompleted   bool          `json:"nodeCompleted,omitempty"`
	ProgressUpdated bool          `json:"progressUpdated,omitempty"`
}
