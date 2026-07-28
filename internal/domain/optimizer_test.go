package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlexStringList_unmarshalArray(t *testing.T) {
	var l FlexStringList
	if err := json.Unmarshal([]byte(`["a","b"]`), &l); err != nil {
		t.Fatal(err)
	}
	if len(l) != 2 || l[0] != "a" {
		t.Fatalf("got %v", l)
	}
}

func TestFlexStringList_unmarshalString(t *testing.T) {
	var l FlexStringList
	if err := json.Unmarshal([]byte(`"单条要点"`), &l); err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 || l[0] != "单条要点" {
		t.Fatalf("got %v", l)
	}
}

func TestFlexStringList_unmarshalObject(t *testing.T) {
	var l FlexStringList
	raw := `{"应提到 interface": "实现多态", "方法集": ["值接收者", "指针接收者"]}`
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		t.Fatal(err)
	}
	if len(l) != 2 {
		t.Fatalf("got %v", l)
	}
}

func TestOptimizeNodeLLMOutput_unmarshalGradingHintsObject(t *testing.T) {
	raw := `{
		"grading_hints": {"及格": "能说出方法集规则", "良好": "能写接口实现"},
		"exercise_ideas": "一道 interface 填空题"
	}`
	var out optimizeNodeLLMOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("should tolerate object grading_hints: %v", err)
	}
	if len(out.GradingHints) != 2 {
		t.Fatalf("grading_hints=%v", out.GradingHints)
	}
	if len(out.ExerciseIdeas) != 1 || out.ExerciseIdeas[0] != "一道 interface 填空题" {
		t.Fatalf("exercise_ideas=%v", out.ExerciseIdeas)
	}
}

func TestMergeEnrichedSpec_fromFlexLists(t *testing.T) {
	base := NodeSpec{Key: "k", Node: "n", CoreConcepts: []string{"a"}}
	enriched := optimizeNodeLLMOutput{
		GradingHints: FlexStringList{"应提到方法集"},
	}
	merged := mergeEnrichedSpec(base, enriched)
	if len(merged.GradingHints) != 1 || merged.GradingHints[0] != "应提到方法集" {
		t.Fatalf("merged=%v", merged.GradingHints)
	}
}

func TestMergeEnrichedSpec_padsEmptyMustTeach(t *testing.T) {
	base := NodeSpec{
		Key: "k", Node: "n",
		CoreConcepts:   []string{"包的导入"},
		CommonMistakes: []string{"混淆相对与绝对路径"},
	}
	enriched := optimizeNodeLLMOutput{
		TeachingBeats: []ConceptBeat{
			{Concept: "包的导入", MustTeach: []string{}},
		},
	}
	merged := mergeEnrichedSpec(base, enriched)
	if len(merged.TeachingBeats) != 1 || countNonEmptyMustTeach(merged.TeachingBeats[0].MustTeach) < MinMustTeachItems {
		t.Fatalf("empty must_teach not padded: %+v", merged.TeachingBeats)
	}
}

func TestMergeEnrichedSpec_keepsSingleMustTeach(t *testing.T) {
	base := NodeSpec{
		Key: "k", Node: "n",
		CoreConcepts: []string{"包的导入"},
	}
	enriched := optimizeNodeLLMOutput{
		TeachingBeats: []ConceptBeat{
			{Concept: "包的导入", MustTeach: []string{"import 语法"}},
		},
	}
	merged := mergeEnrichedSpec(base, enriched)
	if len(merged.TeachingBeats) != 1 || len(merged.TeachingBeats[0].MustTeach) != 1 {
		t.Fatalf("single must_teach should not be padded to 2: %+v", merged.TeachingBeats)
	}
}

func TestDescribePatchBenefits_teachingAndExercise(t *testing.T) {
	before := NodePatchFields{}
	after := NodePatchFields{
		TeachingBeats: []ConceptBeat{{Concept: "c", MustTeach: []string{"a", "b"}}},
		ExerciseIdeas: []string{"写一段 select"},
	}
	benefits := describePatchBenefits(before, after, nil)
	if len(benefits) < 2 {
		t.Fatalf("benefits=%v", benefits)
	}
	joined := strings.Join(benefits, " ")
	if !strings.Contains(joined, "讲解") || !strings.Contains(joined, "练习") {
		t.Fatalf("expected learner-facing benefits, got %v", benefits)
	}
}

func TestBuildOptimizeHeadline(t *testing.T) {
	h := buildOptimizeHeadline([]OptimizePatchItem{
		{Benefits: []string{"补齐讲解节拍与必讲要点，学这一节时教练讲得更聚焦、少跑偏"}},
		{Benefits: []string{"丰富练习题思路，后续出题更贴考点"}},
	})
	if !strings.Contains(h, "2 个知识点") || !strings.Contains(h, "讲解") || !strings.Contains(h, "练习") {
		t.Fatalf("headline=%q", h)
	}
}
