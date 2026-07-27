package scenario

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Hello, World!", "hello-world"},
		{"  Spaced  Out  ", "spaced-out"},
		{"already-slugged", "already-slugged"},
		{"Multi---Dash", "multi-dash"},
		{"2026 Roadmap", "2026-roadmap"},
		{"!!!", ""},
	}

	for _, c := range cases {
		if Slugify(c.in) != c.want {
			t.Error("Slugify returned the wrong value")
		}
	}
}
