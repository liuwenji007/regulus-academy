package domain

import (
	"encoding/json"
	"testing"
)

func TestConceptBeatUnmarshalTolerant(t *testing.T) {
	var spec NodeSpec
	raw := `{
		"key": "k", "node": "n", "layer": "精通",
		"core_concepts": ["a", "b", "c"],
		"teaching_beats": [
			"纯字符串节拍",
			{"concept": "对象节拍", "must_teach": "单字符串要点", "context_type": "workplace"},
			{"concept": "正常节拍", "must_teach": ["要点1", "要点2"]}
		]
	}`
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		t.Fatalf("应容错解析: %v", err)
	}
	if len(spec.TeachingBeats) != 3 {
		t.Fatalf("beats=%d", len(spec.TeachingBeats))
	}
	if spec.TeachingBeats[0].Concept != "纯字符串节拍" {
		t.Fatalf("string beat: %+v", spec.TeachingBeats[0])
	}
	if len(spec.TeachingBeats[1].MustTeach) != 1 || spec.TeachingBeats[1].MustTeach[0] != "单字符串要点" {
		t.Fatalf("string must_teach: %+v", spec.TeachingBeats[1])
	}
	if len(spec.TeachingBeats[2].MustTeach) != 2 {
		t.Fatalf("list must_teach: %+v", spec.TeachingBeats[2])
	}
}

func TestNormalizeTeachingBeats_fallback(t *testing.T) {
	spec := &NodeSpec{
		CoreConcepts:   []string{"导数", "极限"},
		CommonMistakes: []string{"混淆左右导数"},
		DomainKind:     DomainKindAcademic,
	}
	beats := NormalizeTeachingBeats(spec)
	if len(beats) != 2 {
		t.Fatalf("expected 2 beats, got %d", len(beats))
	}
	if beats[0].ContextType != ContextIntuition {
		t.Fatalf("academic default context: %s", beats[0].ContextType)
	}
}

func TestEffectiveFirstExerciseLevel_default(t *testing.T) {
	if got := EffectiveFirstExerciseLevel(&NodeSpec{}); got != ExerciseLevelRecognition {
		t.Fatalf("got %s", got)
	}
}

func TestRequiresApplyExercise(t *testing.T) {
	if RequiresApplyExercise("入门") || RequiresApplyExercise("entry") {
		t.Fatal("entry should not require apply")
	}
	if !RequiresApplyExercise("熟悉") || !RequiresApplyExercise("精通") {
		t.Fatal("intermediate/advanced should require apply")
	}
}

func TestUsesOverviewBegin(t *testing.T) {
	if UsesOverviewBegin(&NodeSpec{CoreConcepts: []string{"a", "b"}}) {
		t.Fatal("2 concepts should not use overview")
	}
	if !UsesOverviewBegin(&NodeSpec{CoreConcepts: []string{"a", "b", "c"}}) {
		t.Fatal("3 concepts should use overview")
	}
}

func TestBeatForConcept(t *testing.T) {
	spec := &NodeSpec{
		CoreConcepts: []string{"goroutine 轻量并发"},
		TeachingBeats: []ConceptBeat{{
			Concept:   "goroutine 轻量并发",
			MustTeach: []string{"go 关键字启动"},
		}},
	}
	b := BeatForConcept(spec, "goroutine")
	if b == nil || len(b.MustTeach) != 1 {
		t.Fatal("expected beat match")
	}
}
