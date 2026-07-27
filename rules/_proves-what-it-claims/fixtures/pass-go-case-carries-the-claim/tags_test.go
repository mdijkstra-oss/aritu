package scenario

import (
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	t.Run("trims surrounding whitespace from each tag", func(t *testing.T) {
		raw := "  go , testing  "
		want := []string{"go", "testing"}

		got := Normalize(raw)

		if !slices.Equal(got, want) {
			t.Errorf("Normalize(%q) = %q; want %q", raw, got, want)
		}
	})

	t.Run("drops entries that are only whitespace", func(t *testing.T) {
		raw := "go, ,testing"
		want := []string{"go", "testing"}

		got := Normalize(raw)

		if !slices.Equal(got, want) {
			t.Errorf("Normalize(%q) = %q; want %q", raw, got, want)
		}
	})
}

func TestNormalizeDropsRepeatedTags(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "repeated next to each other", raw: "go,go,testing", want: []string{"go", "testing"}},
		{name: "repeated later in the list", raw: "go,testing,go", want: []string{"go", "testing"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(tc.raw)

			if !slices.Equal(got, tc.want) {
				t.Errorf("Normalize(%q) = %q; want %q", tc.raw, got, tc.want)
			}
		})
	}
}
