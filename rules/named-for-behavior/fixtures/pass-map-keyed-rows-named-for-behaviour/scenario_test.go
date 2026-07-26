package scenario

import (
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	const (
		base    = 100 * time.Millisecond
		ceiling = time.Second
	)

	cases := map[string]struct {
		attempt int
		want    time.Duration
	}{
		"returns the base delay for the first attempt": {attempt: 0, want: base},
		"doubles the delay for every later attempt":    {attempt: 2, want: 400 * time.Millisecond},
		"caps the delay at the ceiling":                {attempt: 9, want: ceiling},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := NextBackoff(tc.attempt, base, ceiling)

			if got != tc.want {
				t.Fatalf("NextBackoff(%d, %v, %v) = %v; want %v", tc.attempt, base, ceiling, got, tc.want)
			}
		})
	}
}
