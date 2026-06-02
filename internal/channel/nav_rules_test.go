package channel

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestMatchNavigationRules(t *testing.T) {
	ctx := navContext{
		Courses: []storage.DomainSummary{
			{ID: "d1", Name: "Go 并发", Slug: "go-concurrency", NodeTotal: 3},
		},
		FlatNodes: []flatNode{
			{Key: "goroutine_basics", Title: "goroutine 是什么", Layer: "entry"},
			{Key: "channel", Title: "channel 通信", Layer: "intermediate"},
		},
		PendingDomainID: "d1",
	}

	cases := []struct {
		text   string
		action NavAction
	}{
		{"我的课程", NavListCourses},
		{"有哪些课", NavListCourses},
		{"进度怎么样", NavProgress},
		{"接着学", NavContinue},
		{"继续学", NavContinue},
		{"怎么用", NavHelp},
		{"学 Go 并发", NavShowNodes},
		{"打开第1门课", NavShowNodes},
		{"Go 并发第一个节点", NavStartNode},
		{"第2个节点", NavStartNode},
	}
	for _, tc := range cases {
		intent, ok := matchNavigationRules(tc.text, ctx)
		if !ok {
			t.Fatalf("%q: expected match", tc.text)
		}
		if intent.Action != tc.action {
			t.Fatalf("%q: action=%s want %s", tc.text, intent.Action, tc.action)
		}
	}

	if _, ok := matchNavigationRules("什么是 channel", ctx); ok {
		t.Fatal("普通提问不应匹配导航")
	}
	if _, ok := matchNavigationRules("一般go标准项目里需要哪些", ctx); ok {
		t.Fatal("答疑不应误匹配选课导航")
	}
}

func TestMatchNavigationRulesWhileLearning(t *testing.T) {
	ctx := navContext{HasActiveSession: true}
	if _, ok := matchNavigationRulesWhileLearning("一般go标准项目里需要哪些", ctx); ok {
		t.Fatal("学习中答疑不应走导航")
	}
	if intent, ok := matchNavigationRulesWhileLearning("我的课程", ctx); !ok || intent.Action != NavListCourses {
		t.Fatalf("应仍可查看课表: %+v ok=%v", intent, ok)
	}
}

func TestResolveCourseRef(t *testing.T) {
	list := []storage.DomainSummary{
		{ID: "d1", Name: "Go 并发", Slug: "go-concurrency"},
		{ID: "d2", Name: "Rust 入门", Slug: "rust"},
	}
	if id, ok := resolveCourseRef(list, "1"); !ok || id != "d1" {
		t.Fatalf("ordinal: id=%s ok=%v", id, ok)
	}
	if id, ok := resolveCourseRef(list, "go-concurrency"); !ok || id != "d1" {
		t.Fatalf("slug: id=%s ok=%v", id, ok)
	}
	if id, ok := resolveCourseRef(list, "并发"); !ok || id != "d1" {
		t.Fatalf("substring: id=%s ok=%v", id, ok)
	}
}

func TestResolveNodeRef(t *testing.T) {
	nodes := []flatNode{
		{Key: "goroutine_basics", Title: "goroutine 是什么", Layer: "entry"},
		{Key: "channel", Title: "channel 通信", Layer: "intermediate"},
	}
	if key, layer, ok := resolveNodeRef(nodes, "2"); !ok || key != "channel" || layer != "intermediate" {
		t.Fatalf("ordinal: key=%s layer=%s ok=%v", key, layer, ok)
	}
	if key, _, ok := resolveNodeRef(nodes, "channel 通信"); !ok || key != "channel" {
		t.Fatalf("title: key=%s ok=%v", key, ok)
	}
}

func TestMatchesNextSection(t *testing.T) {
	if !matchesNextSection("下一节") {
		t.Fatal("应识别下一节")
	}
	if matchesNextSection("下一节是什么") {
		t.Fatal("疑问句不应触发")
	}
}
