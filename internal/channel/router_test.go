package channel

import (
	"testing"
)

func TestSplitMessage(t *testing.T) {
	short := "hello"
	if parts := SplitMessage(short, 100); len(parts) != 1 || parts[0] != short {
		t.Fatalf("short message: %v", parts)
	}

	long := string(make([]rune, 4000))
	for i := range long {
		// can't assign to string index for runes - use different approach
		_ = i
	}
	longRunes := make([]rune, 4000)
	for i := range longRunes {
		longRunes[i] = '学'
	}
	long = string(longRunes)
	parts := SplitMessage(long, 3500)
	if len(parts) < 2 {
		t.Fatalf("expected split, got %d parts", len(parts))
	}
}

func TestParseCommand(t *testing.T) {
	cases := []struct {
		in, cmd, arg string
	}{
		{"绑定 小明", "bind", "小明"},
		{"课程", "courses", ""},
		{"学习 1", "learn", "1"},
		{"节点 2", "node", "2"},
		{"继续", "continue", ""},
		{"帮助", "help", ""},
		{"什么是 channel", "", "什么是 channel"},
	}
	for _, tc := range cases {
		cmd, arg := parseCommand(tc.in)
		if cmd != tc.cmd || arg != tc.arg {
			t.Errorf("%q => (%q,%q) want (%q,%q)", tc.in, cmd, arg, tc.cmd, tc.arg)
		}
	}
}

func TestParsePositiveInt(t *testing.T) {
	if parsePositiveInt("3") != 3 {
		t.Fatal("expected 3")
	}
	if parsePositiveInt("abc") != 0 {
		t.Fatal("expected 0")
	}
}
