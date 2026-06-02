package agent

import (
	"encoding/json"
	"strings"
)

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		s = s[i:]
	}
	if j := strings.LastIndex(s, "}"); j >= 0 {
		s = s[:j+1]
	}
	return s
}

// sanitizeCoachPlainText 剥离误输出的结构化 JSON，只保留给用户看的正文
func sanitizeCoachPlainText(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return content
	}
	if fb, ok := parseGradeJSONText(content); ok {
		return fb
	}
	if fb, ok := parseMasteryJSONText(content); ok {
		return fb
	}
	return content
}

func parseGradeJSONText(content string) (feedback string, ok bool) {
	raw := extractJSONObject(content)
	var aux map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &aux); err != nil {
		return "", false
	}
	if _, has := aux["passed"]; !has {
		return "", false
	}
	if r, has := aux["feedback"]; has {
		_ = json.Unmarshal(r, &feedback)
	}
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		var passed bool
		_ = json.Unmarshal(aux["passed"], &passed)
		if passed {
			feedback = "回答正确，很好。"
		} else {
			feedback = "这轮还没完全过关，建议再巩固一下。"
		}
	}
	return feedback, true
}

func parseMasteryJSONText(content string) (feedback string, ok bool) {
	raw := extractJSONObject(content)
	var aux map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &aux); err != nil {
		return "", false
	}
	if _, has := aux["ready"]; !has {
		return "", false
	}
	if r, has := aux["feedback"]; has {
		_ = json.Unmarshal(r, &feedback)
	}
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		var ready bool
		_ = json.Unmarshal(aux["ready"], &ready)
		if ready {
			feedback = "掌握不错，可以进入下一节。"
		} else {
			feedback = "还有一些薄弱点建议再巩固一下。"
		}
	}
	return feedback, true
}

func mergeGradeMistakes(out *GradeOutput) {
	if len(out.MistakeConcepts) == 0 && len(out.WeakPoints) > 0 {
		out.MistakeConcepts = out.WeakPoints
	}
}
