package scenario

import "testing"

func TestParseSplitsHostFromPort(t *testing.T) {
	got, err := Parse("db.internal:5432")
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", "db.internal:5432", err)
	}
	if got.Host != "db.internal" || got.Port != 5432 {
		t.Fatalf("Parse(%q) = %+v; want host %q and port %d", "db.internal:5432", got, "db.internal", 5432)
	}
}

func TestParseReadsThePortAfterTheColon(t *testing.T) {
	got, err := Parse("db.internal:5432")
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", "db.internal:5432", err)
	}
	if got.Port != 5432 {
		t.Fatalf("Parse(%q).Port = %d; want %d", "db.internal:5432", got.Port, 5432)
	}
}
