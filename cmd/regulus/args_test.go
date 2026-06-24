package main

import (
	"strings"
	"testing"
)

func TestParseBuildArgs_flagsAfterTopic(t *testing.T) {
	opt, err := parseBuildArgs([]string{"想学 TypeScript", "--coach-root", "/tmp/coach"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.topic != "想学 TypeScript" {
		t.Fatalf("topic=%q", opt.topic)
	}
	if opt.coachRoot != "/tmp/coach" {
		t.Fatalf("coachRoot=%q", opt.coachRoot)
	}
}

func TestParseBuildArgs_flagsBeforeTopic(t *testing.T) {
	opt, err := parseBuildArgs([]string{"--coach-root", "/tmp/coach", "想学 Rust"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.topic != "想学 Rust" {
		t.Fatalf("topic=%q", opt.topic)
	}
	if opt.coachRoot != "/tmp/coach" {
		t.Fatalf("coachRoot=%q", opt.coachRoot)
	}
}

func TestParseBuildArgs_topicMustNotIncludeFlags(t *testing.T) {
	opt, err := parseBuildArgs([]string{"想学 TypeScript", "--coach-root", "/tmp/coach"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(opt.topic, "--coach-root") {
		t.Fatalf("topic polluted: %q", opt.topic)
	}
}

func TestParseSessionMessageArgs(t *testing.T) {
	opt, err := parseSessionMessageArgs([]string{"你好", "--session", "abc123"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.sessionID != "abc123" || opt.text != "你好" {
		t.Fatalf("got session=%q text=%q", opt.sessionID, opt.text)
	}
}
