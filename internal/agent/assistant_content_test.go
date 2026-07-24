package agent

import "testing"

func TestLooksLikeStructuredCoachOutput(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"你好，goroutine 是…", false},
		{`{"passed":true,"feedback":"ok"}`, true},
		{"```json\n{", true},
		{"```JSON\n{", true},
		// 正文中的代码花括号不得误杀流式
		{"可以用 `map[string]int`，例如：\n\n```go\nfunc main() {\n  x := 1\n}\n```", false},
		{"先说明一下：{\"passed\":true}", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := looksLikeStructuredCoachOutput(tc.in); got != tc.want {
			t.Fatalf("in=%q got=%v want=%v", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeCoachPlainText_gradeJSON(t *testing.T) {
	got := sanitizeCoachPlainText(`{"passed":true,"feedback":"回答正确","mistake_concepts":[]}`)
	if got != "回答正确" {
		t.Fatalf("got %q", got)
	}
}
