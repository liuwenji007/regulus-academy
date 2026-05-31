package channel

import (
	"context"
	"errors"
	"testing"
)

type mockAdapter struct {
	name   string
	sent   []string
	failAt int
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Start(context.Context, func(MessageEvent)) error {
	return nil
}
func (m *mockAdapter) SendText(_ context.Context, _ ReplyTarget, text string) error {
	m.sent = append(m.sent, text)
	if m.failAt > 0 && len(m.sent) == m.failAt {
		return errors.New("send failed")
	}
	return nil
}

func TestDeliverRetries(t *testing.T) {
	ad := &mockAdapter{name: "test", failAt: 1}
	Deliver(context.Background(), ad, ReplyTarget{}, []string{"hello"})
	if len(ad.sent) != 2 {
		t.Fatalf("expected 2 send attempts, got %d", len(ad.sent))
	}
}

func TestSplitMessageChunks(t *testing.T) {
	parts := SplitMessage("x"+string(make([]rune, 4000)), 100)
	if len(parts) < 2 {
		t.Fatalf("expected split, got %d parts", len(parts))
	}
}
