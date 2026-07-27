package scenario

import (
	"slices"
	"testing"
)

func TestSearchFindsDocumentsWhoseTextContainedThePluralForm(t *testing.T) {
	idx := &Index{postings: map[string][]int{"gopher": {3, 7}}}

	want := []int{3, 7}
	if got := idx.Search("gophers"); !slices.Equal(got, want) {
		t.Errorf("Search(%q) = %v, want %v", "gophers", got, want)
	}
}
