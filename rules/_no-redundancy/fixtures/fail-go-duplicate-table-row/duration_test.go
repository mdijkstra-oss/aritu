package scenario

import "testing"

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		name    string
		seconds int
		want    string
	}{
		{"seconds only", 45, "45s"},
		{"minutes and seconds", 90, "1m 30s"},
		{"hours and minutes", 7500, "2h 5m"},
		{"ninety seconds", 90, "1m 30s"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDuration(tc.seconds)
			if got != tc.want {
				t.Fatalf("FormatDuration(%d) = %q; want %q", tc.seconds, got, tc.want)
			}
		})
	}
}
