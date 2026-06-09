package agent

import (
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestBeginTaskInstruction_requiresTeachingNotSyllabus(t *testing.T) {
	instr := beginTaskInstruction(nil)
	if !strings.Contains(instr, "开场讲解") {
		t.Fatalf("empty node: %s", instr)
	}
	instr2 := beginTaskInstruction(&domain.NodeSpec{CoreConcepts: []string{"a", "b"}})
	if !strings.Contains(instr2, "分条") || strings.Contains(instr2, "禁止") {
		t.Fatalf("2 concepts: %s", instr2)
	}
	instr3 := beginTaskInstruction(&domain.NodeSpec{CoreConcepts: []string{"a", "b", "c"}})
	if !strings.Contains(instr3, "适度展开") || strings.Contains(instr3, "禁止") {
		t.Fatalf("3 concepts: %s", instr3)
	}
	instr5 := beginTaskInstruction(&domain.NodeSpec{
		CoreConcepts: []string{"a", "b", "c", "d", "e"},
	})
	if !strings.Contains(instr5, "总览") || strings.Contains(instr5, "禁止") {
		t.Fatalf("5 concepts: %s", instr5)
	}
}

func TestRecordBeginExplained_doesNotSeedAllConcepts(t *testing.T) {
	sctx := &storage.SessionContext{}
	node := &domain.NodeSpec{CoreConcepts: []string{"a", "b", "c"}}
	recordBeginExplained(sctx, node)
	if !sctx.OverviewDone {
		t.Fatal("expected OverviewDone")
	}
	if len(sctx.ExplainedConcepts) != 0 {
		t.Fatalf("begin should not seed ExplainedConcepts, got %v", sctx.ExplainedConcepts)
	}
}
