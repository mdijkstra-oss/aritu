package scenario

import "testing"

func TestPullsValuesOutsideThePercentageRangeToTheNearestBound(t *testing.T) {
	cases := []struct {
		name  string
		value int
		want  int
	}{
		{name: "far below the floor", value: -250, want: 0},
		{name: "just below the floor", value: -1, want: 0},
		{name: "at the floor", value: 0, want: 0},
		{name: "inside the range", value: 42, want: 42},
		{name: "at the ceiling", value: 100, want: 100},
		{name: "just above the ceiling", value: 101, want: 100},
		{name: "far above the ceiling", value: 4000, want: 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClampPercentage(tc.value)
			if got != tc.want {
				t.Fatalf("ClampPercentage(%d) = %d; want %d", tc.value, got, tc.want)
			}
		})
	}
}
