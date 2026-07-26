package scenario

import (
	"slices"
	"testing"
)

func TestDeduplicate(t *testing.T) {
	t.Run("drops values that already appeared", func(t *testing.T) {
		values := []string{"go", "go", "rust", "go"}
		want := []string{"go", "rust"}

		got := Deduplicate(values)

		if !slices.Equal(got, want) {
			t.Fatalf("Deduplicate(%q) = %q; want %q", values, got, want)
		}
	})

	t.Run("keeps the first occurrence of each value in its original position", func(t *testing.T) {
		values := []string{"rust", "zig", "go", "zig", "rust"}
		want := []string{"rust", "zig", "go"}

		got := Deduplicate(values)

		if !slices.Equal(got, want) {
			t.Fatalf("Deduplicate(%q) = %q; want %q", values, got, want)
		}
	})

	t.Run("leaves the input slice unchanged", func(t *testing.T) {
		values := []string{"go", "go", "rust"}
		want := []string{"go", "go", "rust"}

		Deduplicate(values)

		if !slices.Equal(values, want) {
			t.Fatalf("Deduplicate mutated its input to %q; want %q", values, want)
		}
	})
}
