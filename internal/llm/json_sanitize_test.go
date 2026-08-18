package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeLLMJSON_controlCharInString(t *testing.T) {
	raw := "{\"points\":[\"hello\x18world\"]}"
	got := extractJSON(raw)
	var out struct {
		Points []string `json:"points"`
	}
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v raw=%q", err, got)
	}
	if len(out.Points) != 1 || out.Points[0] != "hello\x18world" {
		t.Fatalf("points=%q", out.Points)
	}
}

func TestSanitizeLLMJSON_invalidQuoteEscape(t *testing.T) {
	raw := `{"points":["it\'s ok"]}`
	got := extractJSON(raw)
	var out struct {
		Points []string `json:"points"`
	}
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v raw=%q", err, got)
	}
	if len(out.Points) != 1 || out.Points[0] != "it's ok" {
		t.Fatalf("points=%q", out.Points)
	}
}

func TestSanitizeLLMJSON_preservesValidJSON(t *testing.T) {
	raw := `{"a":"say \"hi\"","b":"C:\\tmp","n":"line\nbreak","u":"\u4e2d"}`
	got := extractJSON(raw)
	if got != raw {
		t.Fatalf("valid JSON should be unchanged\n got %s\nwant %s", got, raw)
	}
	var dest map[string]string
	if err := json.Unmarshal([]byte(got), &dest); err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeLLMJSON_keepsPrettyWhitespace(t *testing.T) {
	raw := "{\n  \"ok\": true\n}"
	got := extractJSON(raw)
	var dest struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(got), &dest); err != nil {
		t.Fatal(err)
	}
	if !dest.OK {
		t.Fatal("expected ok=true")
	}
}

func TestSanitizeLLMJSON_unescapedNewlineInString(t *testing.T) {
	raw := "{\"q\":\"line1\nline2\"}"
	got := extractJSON(raw)
	var out struct {
		Q string `json:"q"`
	}
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v got=%q", err, got)
	}
	if out.Q != "line1\nline2" {
		t.Fatalf("q=%q", out.Q)
	}
}

func TestExtractJSON_controlCharAfterFence(t *testing.T) {
	raw := "```json\n{\"points\":[\"Ý\x18Ñ\"]}\n```"
	got := extractJSON(raw)
	var out struct {
		Points []string `json:"points"`
	}
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v got=%q", err, got)
	}
	if len(out.Points) != 1 || !strings.Contains(out.Points[0], "Ý") {
		t.Fatalf("points=%q", out.Points)
	}
}
