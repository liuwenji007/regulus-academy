package agent

import (
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

var exerciseSubmitPromptMarkers = []string{
	"做完后直接把答案发给我",
	"做完直接把答案发给我",
	"做完后把答案发给我",
	"做完把答案发给我",
	"直接把答案发给我",
}

func compactForExercisePrompt(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// looksLikeExerciseSubmitPrompt 助手消息是否已进入「请用户作答」态（含 LLM 措辞变体）。
func looksLikeExerciseSubmitPrompt(content string) bool {
	compact := compactForExercisePrompt(content)
	for _, m := range exerciseSubmitPromptMarkers {
		if strings.Contains(compact, compactForExercisePrompt(m)) {
			return true
		}
	}
	return false
}

func stripExerciseSubmitSuffix(content string) string {
	trimmed := strings.TrimSpace(content)
	for _, m := range exerciseSubmitPromptMarkers {
		if i := strings.LastIndex(trimmed, m); i >= 0 {
			before := strings.TrimSpace(trimmed[:i])
			if before != "" {
				return strings.TrimRight(before, "。，,. \n")
			}
		}
		// 带句号的全角结尾
		if i := strings.LastIndex(trimmed, m+"。"); i >= 0 {
			before := strings.TrimSpace(trimmed[:i])
			if before != "" {
				return strings.TrimRight(before, "。，,. \n")
			}
		}
	}
	return trimmed
}

func (c *Coach) adoptPlainTextExercise(sess *storage.Session, sctx *storage.SessionContext, content string) (*MessageResult, error) {
	question := stripExerciseSubmitSuffix(content)
	if question == "" {
		question = content
	}
	sctx.Exercise = &storage.ExerciseContext{
		Question:     question,
		QuestionType: "short_answer",
		AnswerFormat: "text",
	}
	sess.Phase = "exercise"
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)

	return &MessageResult{
		Role:     "assistant",
		Content:  content,
		Phase:    "exercise",
		Exercise: exerciseMetaFromContext(sctx.Exercise),
	}, nil
}
