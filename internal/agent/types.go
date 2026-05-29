package agent

// ExerciseOutput 出题 JSON
type ExerciseOutput struct {
	Question           string   `json:"question"`
	QuestionType       string   `json:"question_type"`
	ReinforcedConcepts []string `json:"reinforced_concepts"`
}

// GradeOutput 批改 JSON
type GradeOutput struct {
	Passed          bool     `json:"passed"`
	Feedback        string   `json:"feedback"`
	MistakeConcepts []string `json:"mistake_concepts"`
}

// MessageResult API 返回
type MessageResult struct {
	Role            string `json:"role"`
	Content         string `json:"content"`
	Phase           string `json:"phase"`
	NodeCompleted   bool   `json:"nodeCompleted,omitempty"`
	ProgressUpdated bool   `json:"progressUpdated,omitempty"`
}
