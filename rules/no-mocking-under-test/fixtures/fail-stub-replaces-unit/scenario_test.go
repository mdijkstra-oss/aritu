package scenario

import "testing"

type stubNormalizer struct {
	result string
}

var _ Normalizer = stubNormalizer{}

func (s stubNormalizer) Normalize(address string) string { return s.result }

func TestEmailNormalizerLowercasesTheDomainAndKeepsTheLocalPart(t *testing.T) {
	var normalizer Normalizer = stubNormalizer{result: "Ada.Lovelace@example.com"}

	got := normalizer.Normalize("  Ada.Lovelace@EXAMPLE.COM  ")

	if got != "Ada.Lovelace@example.com" {
		t.Errorf("Normalize() = %q, want %q", got, "Ada.Lovelace@example.com")
	}
}
