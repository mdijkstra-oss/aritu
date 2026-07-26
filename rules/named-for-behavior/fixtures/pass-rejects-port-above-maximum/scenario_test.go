package scenario

import "testing"

func TestRejectsPortAboveMaximum(t *testing.T) {
	port, err := ParsePort("65536")

	if err == nil {
		t.Fatalf("ParsePort(%q) = %d, nil; want an error", "65536", port)
	}
	if port != 0 {
		t.Fatalf("ParsePort(%q) = %d; want 0 alongside the error", "65536", port)
	}
}
