package domain

import "testing"

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
