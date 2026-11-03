package domain

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestChunkForRetrieval_paragraphs(t *testing.T) {
	para := strings.Repeat("计费阈值默认为四十二。", 40) // ~440 runes
	text := para + "\n\n" + para + "\n\n# 下一节\n" + para
	chunks := ChunkForRetrieval(text)
	if len(chunks) < 2 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	for _, c := range chunks {
		if n := utf8.RuneCountInString(c); n > 900 {
			t.Fatalf("chunk too long: %d", n)
		}
	}
}

func TestScoreOverlap_identifierAndBigram(t *testing.T) {
	doc := "内部接口 AcmeQuota 的计费阈值默认为 42。"
	if ScoreOverlap("AcmeQuota", doc) < 3 {
		t.Fatal("identifier should score")
	}
	if ScoreOverlap("计费阈值", doc) <= 0 {
		t.Fatal("cjk bigram should score")
	}
	if ScoreOverlap("完全无关的话题", doc) != 0 {
		t.Fatal("unrelated query should be 0")
	}
}

func TestGroundNodes_writesMustTeach(t *testing.T) {
	nodes := map[string]NodeSpec{
		"quota": {
			Key:          "quota",
			Node:         "计费阈值",
			CoreConcepts: []string{"计费阈值"},
		},
	}
	chunks := []storage.SourceChunk{{
		ID:   "c1",
		Text: "计费阈值默认为 42 个请求。超过上限必须拒绝。不要把默认值改成无限制。",
	}}
	updated, bind := GroundNodes(nodes, chunks)
	if bind["c1"] != "quota" {
		t.Fatalf("bind=%v", bind)
	}
	spec := updated["quota"]
	blob := strings.Join(spec.GradingHints, " ")
	for _, b := range spec.TeachingBeats {
		blob += " " + strings.Join(b.MustTeach, " ")
	}
	blob += " " + strings.Join(spec.CommonMistakes, " ")
	if !strings.Contains(blob, "42") && !strings.Contains(blob, "必须拒绝") {
		t.Fatalf("facts not grounded: %+v", spec)
	}
}

func TestSelectExcerpts_prefersNode(t *testing.T) {
	chunks := []storage.SourceChunk{
		{ID: "a", Text: "计费阈值是内部名词。", NodeKey: "quota", Filename: "wiki.md"},
		{ID: "b", Text: "计费阈值出现在别的段落。", NodeKey: "other", Filename: "other.md"},
	}
	cites := SelectExcerpts(chunks, "quota", "计费阈值", 1)
	if len(cites) != 1 || cites[0].Index != 1 || cites[0].Filename != "wiki.md" {
		t.Fatalf("cites=%+v", cites)
	}
	prompt := FormatExcerptPrompt(cites)
	if !strings.Contains(prompt, "[1]") || !strings.Contains(prompt, "wiki.md") {
		t.Fatalf("prompt=%s", prompt)
	}
}
