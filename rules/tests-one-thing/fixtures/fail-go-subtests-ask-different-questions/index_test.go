package scenario

import (
	"slices"
	"testing"
)

func TestIndexAdd(t *testing.T) {
	t.Run("returns the normalized key", func(t *testing.T) {
		got := New().Add("  Release Notes 2026! ", "notes.md")
		if got != "release-notes-2026" {
			t.Errorf("Add(...) = %q, want %q", got, "release-notes-2026")
		}
	})

	t.Run("appends a line to the journal", func(t *testing.T) {
		idx := New()
		idx.Add("Release Notes 2026", "notes.md")

		want := []string{"add release-notes-2026 -> notes.md"}
		if got := idx.Journal(); !slices.Equal(got, want) {
			t.Errorf("Journal() = %v, want %v", got, want)
		}
	})
}
