package ingest

import (
	"strings"
	"testing"
)

func TestNormalizeText_stripsControlChars(t *testing.T) {
	got := normalizeText("QUALITY\x18CAN\n\nWORKFLOW")
	if strings.ContainsRune(got, 0x18) {
		t.Fatalf("control char remains: %q", got)
	}
	if !strings.Contains(got, "QUALITY") || !strings.Contains(got, "CAN") || !strings.Contains(got, "WORKFLOW") {
		t.Fatalf("lost content: %q", got)
	}
}

func TestNormalizeText_keepsNewlinesAndTabs(t *testing.T) {
	got := normalizeText("a\tb\nc")
	if got != "a\tb\nc" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeText_collapsesBlankLines(t *testing.T) {
	got := normalizeText("a\n\n\n\nb")
	if got != "a\n\nb" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeText_dropsInvalidUTF8(t *testing.T) {
	got := normalizeText("ok" + string([]byte{0xff, 0xfe}) + "end")
	if strings.ContainsRune(got, '\uFFFD') {
		t.Fatalf("replacement char leaked: %q", got)
	}
	if !strings.Contains(got, "ok") || !strings.Contains(got, "end") {
		t.Fatalf("got %q", got)
	}
}
