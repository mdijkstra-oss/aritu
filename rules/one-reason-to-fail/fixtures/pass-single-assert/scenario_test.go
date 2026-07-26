package scenario

import "testing"

func TestReturnsTheFallbackWhenTheKeyIsMissing(t *testing.T) {
	settings := map[string]string{"region": "eu-west"}

	got := Lookup(settings, "timeout", "30s")

	if got != "30s" {
		t.Fatalf("Lookup(settings, %q, %q) = %q; want %q", "timeout", "30s", got, "30s")
	}
}
