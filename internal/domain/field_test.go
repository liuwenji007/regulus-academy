package domain

import (
	"context"
	"testing"
)

func TestResolveFieldChoiceUsesAllowedName(t *testing.T) {
	allowed := AllowedFields(nil)
	got, created := ResolveFieldChoice("编程语言", "AI 与数据", 0.9, allowed)
	if got != "编程语言" || created {
		t.Fatalf("got %q created=%v", got, created)
	}
}

func TestResolveFieldChoiceCreatesWhenUnsureAndRoom(t *testing.T) {
	allowed := AllowedFields(nil)
	got, created := ResolveFieldChoice("烹饪", "人文与通识", 0.4, allowed)
	if got != "烹饪" || !created {
		t.Fatalf("got %q created=%v", got, created)
	}
}

func TestResolveFieldChoiceRejectsConfidentNewName(t *testing.T) {
	allowed := AllowedFields(nil)
	got, created := ResolveFieldChoice("烹饪", "人文与通识", 0.9, allowed)
	if got != "人文与通识" || created {
		t.Fatalf("got %q created=%v", got, created)
	}
}

func TestResolveFieldChoiceRejectsWhenFull(t *testing.T) {
	allowed := AllowedFields([]string{"设计", "商业"})
	if len(allowed) != MaxDomainFields {
		t.Fatalf("allowed = %d, want %d", len(allowed), MaxDomainFields)
	}
	got, created := ResolveFieldChoice("烹饪", "人文与通识", 0.2, allowed)
	if got != "人文与通识" || created {
		t.Fatalf("got %q created=%v", got, created)
	}
}

func TestResolveFieldChoiceEmptyFallsBack(t *testing.T) {
	got, created := ResolveFieldChoice("", "", 0, AllowedFields(nil))
	if got != PresetFields[0] || created {
		t.Fatalf("got %q created=%v", got, created)
	}
}

func TestClassifyDomainFieldMock(t *testing.T) {
	mock := &mockLLM{jsonReply: `{"field":"编程语言","nearest":"编程语言","confidence":0.92,"reason":"Go 是编程语言"}`}
	got, err := ClassifyDomainField(context.Background(), mock, "Go 语言", "go", []string{"并发", "语法"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field != "编程语言" || got.Created {
		t.Fatalf("%+v", got)
	}
}

func TestClassifyDomainFieldEmptyJSON(t *testing.T) {
	mock := &mockLLM{jsonReply: `{}`}
	got, err := ClassifyDomainField(context.Background(), mock, "随便", "misc", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field != PresetFields[0] || got.Created {
		t.Fatalf("%+v", got)
	}
}

func TestClassifyDomainFieldWithoutClient(t *testing.T) {
	got, err := ClassifyDomainField(context.Background(), nil, "Go", "go", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field != PresetFields[0] {
		t.Fatalf("%+v", got)
	}
}
