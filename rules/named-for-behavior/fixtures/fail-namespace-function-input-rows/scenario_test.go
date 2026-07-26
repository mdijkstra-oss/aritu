package scenario

import "testing"

func TestParseClock(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		wantHour   int
		wantMinute int
		wantError  bool
	}{
		{name: "midnight", raw: "00:00"},
		{name: "quarter past nine", raw: "09:15", wantHour: 9, wantMinute: 15},
		{name: "no colon", raw: "0915", wantError: true},
		{name: "hour of 24", raw: "24:00", wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hour, minute, err := ParseClock(tc.raw)

			if (err != nil) != tc.wantError {
				t.Fatalf("ParseClock(%q) error = %v; want an error: %t", tc.raw, err, tc.wantError)
			}
			if hour != tc.wantHour || minute != tc.wantMinute {
				t.Fatalf("ParseClock(%q) = %d, %d; want %d, %d", tc.raw, hour, minute, tc.wantHour, tc.wantMinute)
			}
		})
	}
}
