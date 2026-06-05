package agent

import (
	"encoding/json"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func parseExerciseJSONText(content string) (ExerciseOutput, bool) {
	content = strings.TrimSpace(content)
	if i := strings.Index(content, "{"); i >= 0 {
		content = content[i:]
	}
	if j := strings.LastIndex(content, "}"); j >= 0 {
		content = content[:j+1]
	}
	var out ExerciseOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return ExerciseOutput{}, false
	}
	return out, strings.TrimSpace(out.Question) != ""
}

func (c *Coach) adoptExerciseOutput(sess *storage.Session, sctx *storage.SessionContext, out ExerciseOutput) (*MessageResult, error) {
	sctx.Exercise = BuildExerciseContext(out)
	if node, err := c.registry.GetNode(c.store, sess.DomainID, sess.DomainSlug, sess.NodeKey); err == nil && node != nil {
		RecordExerciseTested(sctx, node.CoreConcepts, sctx.Exercise.ReinforcedConcepts)
	}
	sess.Phase = "exercise"
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)

	userContent := strings.TrimSpace(out.Question) + "\n\n做完后直接把答案发给我。"
	return &MessageResult{
		Role:     "assistant",
		Content:  userContent,
		Phase:    "exercise",
		Exercise: exerciseMetaFromContext(sctx.Exercise),
	}, nil
}
