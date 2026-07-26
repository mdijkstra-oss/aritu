package scenario

import "testing"

func TestSlugifyCollapsesPunctuationIntoSingleDashes(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  string
	}{
		{name: "words separated by a space", title: "Hello World", want: "hello-world"},
		{name: "punctuation next to a space", title: "Hello, World!", want: "hello-world"},
		{name: "surrounding whitespace", title: "  Go 1.25 Release  ", want: "go-1-25-release"},
		{name: "only punctuation", title: "---", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Slugify(tc.title)
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}
