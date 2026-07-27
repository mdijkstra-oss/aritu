package scenario

import (
	"strings"
	"testing"
)

func TestParseReturnsTheHostAndPortOfAValidAddress(t *testing.T) {
	got, err := Parse("cache.internal:6379")

	if err != nil {
		t.Fatalf("Parse(\"cache.internal:6379\") returned error %v, want no error", err)
	}
	if got.Host != "cache.internal" {
		t.Errorf("host = %q, want %q", got.Host, "cache.internal")
	}
	if got.Port != 6379 {
		t.Errorf("port = %d, want %d", got.Port, 6379)
	}
}

func TestParseRejectsAPortAboveTheMaximum(t *testing.T) {
	_, err := Parse("cache.internal:70000")

	if err == nil {
		t.Fatal("Parse(\"cache.internal:70000\") accepted port 70000, want an out-of-range error")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error = %q, want it to name the accepted range", err)
	}
}
