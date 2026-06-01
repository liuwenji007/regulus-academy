package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestIsSubtopicOfGoConcurrency(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()

	if !r.IsSubtopicOf("go-concurrency", "go-language") {
		t.Fatal("go-concurrency should be subtopic of go-language")
	}
	if !r.IsSubtopicOf("go-concurrency", "go") {
		t.Fatal("go-concurrency should be subtopic of go")
	}
	if r.IsSubtopicOf("go-language", "go-concurrency") {
		t.Fatal("go-language should not be subtopic of go-concurrency")
	}
}

func TestFindRelatedDomainExistingSubtopic(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	existing := []storage.DomainSummary{
		{ID: "d1", Name: "Go 并发", Slug: "go-concurrency"},
	}
	rel, err := r.FindRelatedDomain(existing, "go-language", "Go 语言")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil || rel.Kind != RelationExistingSubtopic {
		t.Fatalf("got %+v", rel)
	}
}

func TestFindRelatedDomainNewIsSubtopic(t *testing.T) {
	chdirRepo(t)
	r := NewRegistry()
	existing := []storage.DomainSummary{
		{ID: "d2", Name: "Go 语言", Slug: "go-language"},
	}
	rel, err := r.FindRelatedDomain(existing, "go-concurrency", "Go 并发")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil || rel.Kind != RelationNewIsSubtopic {
		t.Fatalf("got %+v", rel)
	}
}

func TestTopicRoot(t *testing.T) {
	if TopicRoot("go-language") != "go" {
		t.Fatal()
	}
	if TopicRoot("rust") != "rust" {
		t.Fatal()
	}
}
