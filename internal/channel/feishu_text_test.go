package channel

import "testing"

func TestParseFeishuText(t *testing.T) {
	got := parseFeishuText("text", `{"text":"@_user_1 你好"}`)
	if got != "你好" {
		t.Fatalf("text mention strip: got %q", got)
	}
	got = parseFeishuText("post", `{"content":[[{"tag":"text","text":"帮助"}]]}`)
	if got != "帮助" {
		t.Fatalf("post parse: got %q", got)
	}
}
