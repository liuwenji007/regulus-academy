package llm

import "strings"

// extractJSON 从模型回复中剥离 markdown 围栏并尽量截取最外层 JSON 对象/数组。
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = stripMarkdownCodeFence(s)
	s = strings.TrimSpace(s)
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		return s
	}
	s = s[start:]
	endByte := byte('}')
	if s[0] == '[' {
		endByte = ']'
	}
	if end := strings.LastIndexByte(s, endByte); end > 0 {
		s = s[:end+1]
	}
	return strings.TrimSpace(s)
}

func stripMarkdownCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if idx := strings.Index(s, "\n"); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSpace(strings.TrimSuffix(s, "```"))
	}
	return s
}
