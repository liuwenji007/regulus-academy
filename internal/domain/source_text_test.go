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

func TestChunkForRetrievalWithOutline_sections(t *testing.T) {
	text := "# 计费阈值\n计费阈值默认为 42。\n\n# 缓存一致性\n缓存写入后必须失效。"
	outline := &DistillOutline{Sections: []DistillSection{
		{Heading: "计费阈值", Concepts: []string{"计费阈值"}},
		{Heading: "缓存一致性", Concepts: []string{"缓存失效"}},
	}}
	drafts := ChunkForRetrievalWithOutline(text, outline)
	if len(drafts) < 2 {
		t.Fatalf("drafts=%d", len(drafts))
	}
	var sawQuota, sawCache bool
	for _, d := range drafts {
		if strings.Contains(d.Text, "42") {
			sawQuota = true
			if len(d.Concepts) != 1 || d.Concepts[0] != "计费阈值" {
				t.Fatalf("quota concepts=%v text=%s", d.Concepts, d.Text)
			}
		}
		if strings.Contains(d.Text, "必须失效") {
			sawCache = true
			if len(d.Concepts) != 1 || d.Concepts[0] != "缓存失效" {
				t.Fatalf("cache concepts=%v", d.Concepts)
			}
		}
	}
	if !sawQuota || !sawCache {
		t.Fatalf("drafts=%+v", drafts)
	}
}

func TestChunkForRetrievalWithOutline_shortHeadingDoesNotStick(t *testing.T) {
	text := "内部接口设计需要先定义边界。\n\n缓存写入后必须失效。"
	outline := &DistillOutline{Sections: []DistillSection{
		{Heading: "接口", Concepts: []string{"接口"}},
		{Heading: "缓存一致性", Concepts: []string{"缓存失效"}},
	}}
	drafts := ChunkForRetrievalWithOutline(text, outline)
	for _, d := range drafts {
		if strings.Contains(d.Text, "必须失效") {
			for _, c := range d.Concepts {
				if c == "接口" {
					t.Fatalf("short heading leaked: %+v", d)
				}
			}
			return
		}
	}
	t.Fatalf("missing cache paragraph: %+v", drafts)
}

func TestChunkForRetrievalWithOutline_conceptFallback(t *testing.T) {
	text := "计费阈值默认为 42。\n\n缓存写入后必须失效。"
	outline := &DistillOutline{Sections: []DistillSection{
		{Heading: "配额策略", Concepts: []string{"计费阈值"}},
		{Heading: "缓存一致性", Concepts: []string{"缓存失效"}},
	}}
	drafts := ChunkForRetrievalWithOutline(text, outline)
	var sawQuota, sawCache bool
	for _, d := range drafts {
		if strings.Contains(d.Text, "42") && containsConcept(d.Concepts, "计费阈值") {
			sawQuota = true
		}
		if strings.Contains(d.Text, "必须失效") && containsConcept(d.Concepts, "缓存失效") {
			sawCache = true
		}
	}
	if !sawQuota || !sawCache {
		t.Fatalf("drafts=%+v", drafts)
	}
}

func containsConcept(concepts []string, want string) bool {
	for _, c := range concepts {
		if c == want {
			return true
		}
	}
	return false
}

func TestAssignChunks_conceptBoost(t *testing.T) {
	nodes := map[string]NodeSpec{
		"cache": {Key: "cache", Node: "缓存一致性", CoreConcepts: []string{"缓存失效"}},
		"other": {Key: "other", Node: "无关主题", CoreConcepts: []string{"无关主题"}},
	}
	chunks := []storage.SourceChunk{{
		ID:       "c1",
		Text:     "这段正文没有直接出现节点标题里的词。",
		Concepts: []string{"缓存失效"},
	}}
	_, bind := GroundNodes(nodes, chunks)
	if bind["c1"] != "cache" {
		t.Fatalf("bind=%v", bind)
	}
}

func TestSelectExcerpts_conceptBoost(t *testing.T) {
	chunks := []storage.SourceChunk{
		{ID: "a", Text: "一段几乎不相关的说明。", Concepts: []string{"计费阈值"}, Filename: "a.md"},
		{ID: "b", Text: "另一段说明。", Filename: "b.md"},
	}
	cites := SelectExcerpts(chunks, "", "计费阈值", 1)
	if len(cites) != 1 || cites[0].Filename != "a.md" {
		t.Fatalf("cites=%+v", cites)
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
