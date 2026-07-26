package scenario

import "testing"

func TestParseAddress(t *testing.T) {
	host, port, err := ParseAddress("localhost:8080")

	if err != nil {
		t.Fatalf("ParseAddress(%q) returned error %v", "localhost:8080", err)
	}
	if host != "localhost" {
		t.Fatalf("host = %q; want %q", host, "localhost")
	}
	if port != "8080" {
		t.Fatalf("port = %q; want %q", port, "8080")
	}
}
