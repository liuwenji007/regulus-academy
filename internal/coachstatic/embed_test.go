package coachstatic

import "testing"

func TestEmbeddedProtocol(t *testing.T) {
	b, err := ReadFile("protocol.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 50 {
		t.Fatal("protocol.md too short")
	}
}
