package agent

import "testing"

func TestNormalizeConcept(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"idempotent", "幂等"},
		{"幂等性", "幂等"},
		{"  Idempotency  ", "幂等"},
		{"hash table", "哈希表"},
		{"HTTP", "HTTP"},
		{"", ""},
	}
	for _, c := range cases {
		got := NormalizeConcept(c.in)
		if got != c.want {
			t.Errorf("NormalizeConcept(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestConceptsLooselyMatch(t *testing.T) {
	if !ConceptsLooselyMatch("幂等", "幂等性") {
		t.Fatal("expected match")
	}
	if !ConceptsLooselyMatch("idempotent", "幂等") {
		t.Fatal("expected synonym match")
	}
	if ConceptsLooselyMatch("缓存", "哈希") {
		t.Fatal("should not match")
	}
}
