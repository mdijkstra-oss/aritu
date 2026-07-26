package scenario

import (
	"slices"
	"testing"
)

func TestNormalizeTags2(t *testing.T) {
	got := NormalizeTags([]string{"Go", "TESTING"})
	want := []string{"go", "testing"}

	if !slices.Equal(got, want) {
		t.Fatalf("NormalizeTags(%q) = %q; want %q", []string{"Go", "TESTING"}, got, want)
	}
}
