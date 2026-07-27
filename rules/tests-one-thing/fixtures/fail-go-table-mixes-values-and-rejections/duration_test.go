package scenario

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"seconds", "90s", 90, false},
		{"minutes", "5m", 300, false},
		{"hours", "2h", 7200, false},
		{"empty input", "", 0, true},
		{"unknown unit", "10x", 0, true},
		{"negative amount", "-3s", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
