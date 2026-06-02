package agent

import "testing"

func TestParseGradeJSONText(t *testing.T) {
	raw := `{"phase":"grade","passed":false,"feedback":"依赖顺序还没讲清","weak_points":["任务依赖排序"]}`
	fb, ok := parseGradeJSONText(raw)
	if !ok || fb != "依赖顺序还没讲清" {
		t.Fatalf("parseGradeJSONText=%q ok=%v", fb, ok)
	}
}

func TestSanitizeCoachPlainTextGrade(t *testing.T) {
	raw := `{"passed":false,"feedback":"再想想 channel 的关闭时机"}`
	got := sanitizeCoachPlainText(raw)
	if got != "再想想 channel 的关闭时机" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeCoachPlainTextMastery(t *testing.T) {
	raw := `{"ready":false,"feedback":"还有缺口","gap_concepts":["x"]}`
	got := sanitizeCoachPlainText(raw)
	if got != "还有缺口" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeGradeMistakes(t *testing.T) {
	out := GradeOutput{WeakPoints: []string{"a", "b"}}
	mergeGradeMistakes(&out)
	if len(out.MistakeConcepts) != 2 {
		t.Fatalf("mistakes=%v", out.MistakeConcepts)
	}
}
