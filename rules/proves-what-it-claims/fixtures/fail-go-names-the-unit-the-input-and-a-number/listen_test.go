package scenario

import "testing"

func TestParseListenAddress(t *testing.T) {
	got, err := ParseListenAddress("example.com:8080")
	if err != nil {
		t.Fatalf("ParseListenAddress(%q) returned error: %v", "example.com:8080", err)
	}
	if got.Host != "example.com" {
		t.Errorf("ParseListenAddress(%q).Host = %q; want %q", "example.com:8080", got.Host, "example.com")
	}
	if got.Port != 8080 {
		t.Errorf("ParseListenAddress(%q).Port = %d; want %d", "example.com:8080", got.Port, 8080)
	}
}

func TestWithEmptyHost(t *testing.T) {
	got, err := ParseListenAddress(":9000")
	if err != nil {
		t.Fatalf("ParseListenAddress(%q) returned error: %v", ":9000", err)
	}
	if got.Host != "0.0.0.0" {
		t.Errorf("ParseListenAddress(%q).Host = %q; want %q", ":9000", got.Host, "0.0.0.0")
	}
}

func TestParseListenAddress2(t *testing.T) {
	_, err := ParseListenAddress("example.com:70000")
	if err == nil {
		t.Fatalf("ParseListenAddress(%q) returned no error; want one", "example.com:70000")
	}
}
