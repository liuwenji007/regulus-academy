package ingest

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestTopicHintFromFilename(t *testing.T) {
	got := topicHintFromFilename("/tmp/Palantir FDE 八大经典案例.pdf")
	if got != "Palantir FDE 八大经典案例" {
		t.Fatalf("got %q", got)
	}
	if topicHintFromFilename("") != "" {
		t.Fatal("empty should stay empty")
	}
}

func TestLooksLikeEncodedGarbage_caesarShift(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("XPSLGMPX VBMJUZ DBO DPPQ HFOU ")
	}
	if !looksLikeEncodedGarbage(b.String()) {
		t.Fatal("caesar-shifted tokens should be flagged")
	}
}

func TestLooksLikeEncodedGarbage_englishOK(t *testing.T) {
	text := strings.Repeat("Palantir Forward Deployed Engineers ship products with customers. ", 6)
	if looksLikeEncodedGarbage(text) {
		t.Fatal("normal English should pass")
	}
}

func TestLooksLikeEncodedGarbage_chineseOK(t *testing.T) {
	text := strings.Repeat("Palantir FDE 八大经典案例深度拆解报告讲述前线部署工程师如何交付。", 4)
	if looksLikeEncodedGarbage(text) {
		t.Fatal("Chinese report should pass")
	}
}

func TestCheckExtractedText_rejectsGarbage(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("XPSLGMPX VBMJUZ DBO DPPQ HFOU ")
	}
	err := checkExtractedText(b.String(), 8)
	if err == nil {
		t.Fatal("garbage should block course build")
	}
}

func TestCheckExtractedText_rejectsSparsePages(t *testing.T) {
	err := checkExtractedText("ab", 10)
	if err == nil {
		t.Fatal("sparse extract should block course build")
	}
}

func TestCheckExtractedText_shortSinglePageOK(t *testing.T) {
	if err := checkExtractedText("Palantir FDE Hello", 1); err != nil {
		t.Fatal(err)
	}
}

func TestFromPDFBytes_extractsHelveticaText(t *testing.T) {
	data := minimalPDF("Palantir FDE Hello")
	src, err := FromPDFBytes(context.Background(), data, "Palantir FDE 八大经典案例.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if src.Kind != KindPDF {
		t.Fatalf("kind=%s", src.Kind)
	}
	if src.TopicHint() != "Palantir FDE 八大经典案例" {
		t.Fatalf("hint=%q", src.TopicHint())
	}
	if !strings.Contains(src.Text, "Palantir") || !strings.Contains(src.Text, "Hello") {
		t.Fatalf("text=%q", src.Text)
	}
	if src.Meta.PageCount != 1 {
		t.Fatalf("pages=%d", src.Meta.PageCount)
	}
}

func TestFromPDFBytes_rejectsEmpty(t *testing.T) {
	_, err := FromPDFBytes(context.Background(), nil, "x.pdf")
	if err == nil {
		t.Fatal("expected error")
	}
}

func minimalPDF(shown string) []byte {
	shown = strings.ReplaceAll(shown, `\`, `\\`)
	shown = strings.ReplaceAll(shown, `(`, `\(`)
	shown = strings.ReplaceAll(shown, `)`, `\)`)
	stream := "BT /F1 12 Tf 72 720 Td (" + shown + ") Tj ET\n"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, body := range objs {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xrefAt := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objs)+1)
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xrefAt)
	return []byte(b.String())
}
