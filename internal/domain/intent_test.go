package domain

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

type mockLLM struct {
	jsonReply string
}

func (m *mockLLM) Configured() bool { return true }
func (m *mockLLM) Name() string     { return "mock" }
func (m *mockLLM) Model() string    { return "mock" }
func (m *mockLLM) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return m.jsonReply, nil
}
func (m *mockLLM) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	return m.Chat(ctx, messages)
}
func (m *mockLLM) ChatJSON(ctx context.Context, messages []llm.Message, temp float64, dest any) error {
	return json.Unmarshal([]byte(m.jsonReply), dest)
}
func (m *mockLLM) Ping(ctx context.Context) error { return nil }

func TestParseIntentRuleMatch(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	res, err := r.ParseIntent(context.Background(), &mockLLM{}, "Go 并发")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != SourceSkillPack || res.Slug != "go-concurrency" {
		t.Fatalf("got %+v", res)
	}
}

func TestParseIntentLLMRust(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	mock := &mockLLM{jsonReply: `{"slug":"rust","displayName":"Rust","confidence":0.9,"reason":"用户想学 Rust 编程"}`}
	res, err := r.ParseIntent(context.Background(), mock, "rust")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != SourceGenerated || res.Slug != "rust" {
		t.Fatalf("got %+v", res)
	}
}

func TestParseIntentLLMGoTopic(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	mock := &mockLLM{jsonReply: `{"slug":"go-concurrency","displayName":"Go 并发","confidence":0.95,"reason":"用户想学习 goroutine"}`}
	res, err := r.ParseIntent(context.Background(), mock, "我想学 goroutine")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != SourceSkillPack || res.Slug != "go-concurrency" {
		t.Fatalf("got %+v", res)
	}
}

func TestSlugify(t *testing.T) {
	if Slugify("Rust") != "rust" {
		t.Fatalf("got %q", Slugify("Rust"))
	}
	if Slugify("  Agent Basics  ") != "agent-basics" {
		t.Fatalf("got %q", Slugify("  Agent Basics  "))
	}
}

func chdirRepo(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			if err := os.Chdir(d); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(wd) })
			return
		}
	}
	t.Fatal("找不到 regulus-coach")
}
