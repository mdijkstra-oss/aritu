package scenario

import "testing"

func TestSplitsAWellFormedAddressIntoHostAndPort(t *testing.T) {
	const raw = "db.internal:5432"

	got, err := ParseAddress(raw)

	if err != nil {
		t.Fatalf("ParseAddress(%q) returned error %v; want none", raw, err)
	}
	if got.Host != "db.internal" {
		t.Errorf("ParseAddress(%q).Host = %q; want %q", raw, got.Host, "db.internal")
	}
	if got.Port != 5432 {
		t.Errorf("ParseAddress(%q).Port = %d; want %d", raw, got.Port, 5432)
	}
}
