package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPersonalizeJSONExample_usesRealKeys(t *testing.T) {
	briefs := []nodeBriefItem{
		{Key: "goroutines", Title: "Goroutine", Layer: "入门"},
		{Key: "channels", Title: "Channel", Layer: "熟悉"},
	}
	ex := personalizeJSONExample(briefs)
	if strings.Contains(ex, "key1") || strings.Contains(ex, "key2") {
		t.Fatalf("should not use placeholder keys: %s", ex)
	}
	if !strings.Contains(ex, `"goroutines"`) || !strings.Contains(ex, `"channels"`) {
		t.Fatalf("should include real keys: %s", ex)
	}
	var parsed personalizeLLMOutput
	if err := json.Unmarshal([]byte(ex), &parsed); err != nil {
		t.Fatalf("example should be valid JSON: %v\n%s", err, ex)
	}
}

func TestDefaultPersonalizeSelection(t *testing.T) {
	briefs := []nodeBriefItem{
		{Key: "a"}, {Key: "b"}, {Key: "c"}, {Key: "d"},
	}
	got := defaultPersonalizeSelection(briefs, 3)
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("got %v", got)
	}
}
