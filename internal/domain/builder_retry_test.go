package domain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

type seqMockLLM struct {
	replies []string
	n       int
}

func (m *seqMockLLM) Configured() bool { return true }
func (m *seqMockLLM) Name() string     { return "seq" }
func (m *seqMockLLM) Model() string    { return "seq" }
func (m *seqMockLLM) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	if m.n >= len(m.replies) {
		return m.replies[len(m.replies)-1], nil
	}
	r := m.replies[m.n]
	m.n++
	return r, nil
}
func (m *seqMockLLM) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	return m.Chat(ctx, messages)
}
func (m *seqMockLLM) ChatJSON(ctx context.Context, messages []llm.Message, temp float64, dest any) error {
	body, err := m.Chat(ctx, messages)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(body), dest)
}
func (m *seqMockLLM) Ping(ctx context.Context) error { return nil }

func TestTreeBuilderBuild_retriesOnValidationError(t *testing.T) {
	t.Setenv("REGULUS_TREE_CRITIQUE", "0")
	var bad buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &bad); err != nil {
		t.Fatal(err)
	}
	delete(bad.Layers, "advanced")
	badJSON, _ := json.Marshal(bad)

	builder := NewTreeBuilder(NewRegistry())
	intent := IntentResult{Slug: "rust", DisplayName: "Rust", Source: SourceGenerated, ScopeBreadth: ScopeModerate}
	mock := &seqMockLLM{replies: []string{string(badJSON), sampleTreeJSON}}
	tree, nodes, err := builder.Build(context.Background(), mock, intent, "rust", "")
	if err != nil {
		t.Fatal(err)
	}
	if mock.n < 2 {
		t.Fatalf("expected retry, calls=%d", mock.n)
	}
	if len(tree.Layers) != 3 || len(nodes) != 8 {
		t.Fatalf("layers=%d nodes=%d", len(tree.Layers), len(nodes))
	}
}
