package scenario

import "testing"

func TestLabelForColumnIndex(t *testing.T) {
	cases := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{25, "Z"},
		{26, "AA"},
		{51, "AZ"},
		{701, "ZZ"},
		{702, "AAA"},
	}

	for _, c := range cases {
		if got := Label(c.index); got != c.want {
			t.Errorf("Label(%d) = %q, want %q", c.index, got, c.want)
		}
	}
}
