package scenario

import (
	"slices"
	"testing"
)

func TestRunningTotals(t *testing.T) {
	cases := []struct {
		name   string
		values []int
		want   []int
	}{
		{name: "case 1", values: []int{1, 2, 3}, want: []int{1, 3, 6}},
		{name: "case 2", values: []int{5}, want: []int{5}},
		{name: "case 3", values: []int{2, -2, 4}, want: []int{2, 0, 4}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RunningTotals(tc.values)

			if !slices.Equal(got, tc.want) {
				t.Fatalf("RunningTotals(%v) = %v; want %v", tc.values, got, tc.want)
			}
		})
	}
}
