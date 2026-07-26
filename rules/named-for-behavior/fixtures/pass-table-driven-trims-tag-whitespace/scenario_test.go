package scenario

import (
	"slices"
	"testing"
)

func TestTrimsSurroundingWhitespaceFromEachTag(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want []string
	}{
		{name: "leading spaces", tags: []string{"  go"}, want: []string{"go"}},
		{name: "trailing tab", tags: []string{"go\t"}, want: []string{"go"}},
		{name: "both sides", tags: []string{" \ngo \n"}, want: []string{"go"}},
		{name: "every element", tags: []string{" go ", " testing "}, want: []string{"go", "testing"}},
		{name: "already trimmed", tags: []string{"go"}, want: []string{"go"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TrimTagWhitespace(tc.tags)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("TrimTagWhitespace(%q) = %q; want %q", tc.tags, got, tc.want)
			}
		})
	}
}
