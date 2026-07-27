package scenario

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", raw: "10s", want: 10 * time.Second},
		{name: "minutes", raw: "5m", want: 5 * time.Minute},
		{name: "hours and minutes", raw: "1h30m", want: time.Hour + 30*time.Minute},
		{name: "no unit", raw: "45", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) returned no error; want one", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %v; want %v", tc.raw, got, tc.want)
			}
		})
	}
}
