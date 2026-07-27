package scenario

import (
	"errors"
	"reflect"
	"testing"
)

func TestOffsetsPlacesBlocksBackToBackUntilCapacityRunsOut(t *testing.T) {
	cases := []struct {
		name     string
		sizes    []int
		capacity int
		want     []int
		wantErr  error
	}{
		{
			name:     "nothing to place",
			sizes:    nil,
			capacity: 10,
			want:     []int{},
		},
		{
			name:     "a single block starts at the front",
			sizes:    []int{4},
			capacity: 10,
			want:     []int{0},
		},
		{
			name:     "each block starts where the previous one ended",
			sizes:    []int{3, 4, 2},
			capacity: 10,
			want:     []int{0, 3, 7},
		},
		{
			name:     "blocks filling the capacity exactly",
			sizes:    []int{5, 5},
			capacity: 10,
			want:     []int{0, 5},
		},
		{
			name:     "one byte more than the capacity holds",
			sizes:    []int{6, 5},
			capacity: 10,
			wantErr:  ErrOverCapacity,
		},
		{
			name:     "a negative block size",
			sizes:    []int{3, -1},
			capacity: 10,
			wantErr:  ErrNegativeSize,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Offsets(tc.sizes, tc.capacity)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Offsets(%v, %d) error = %v, want %v", tc.sizes, tc.capacity, err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Offsets(%v, %d) = %v, want %v", tc.sizes, tc.capacity, got, tc.want)
			}
		})
	}
}
