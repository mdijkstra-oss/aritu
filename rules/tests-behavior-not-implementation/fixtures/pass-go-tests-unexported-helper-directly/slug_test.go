package scenario

import "testing"

func TestCollapseSeparatorsReducesEachRunOfPunctuationToOneHyphen(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "comma and space between two words", in: "hello, world", want: "hello-world"},
		{name: "leading, trailing and doubled spaces", in: "  spaced  out  ", want: "spaced-out"},
		{name: "a run of hyphens", in: "a---b", want: "a-b"},
		{name: "punctuation with no letters or digits", in: "!!!", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := collapseSeparators(tc.in); got != tc.want {
				t.Errorf("collapseSeparators(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
