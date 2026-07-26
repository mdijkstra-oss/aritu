package vote

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	errBoom := errors.New("boom")

	tests := []struct {
		name    string
		rounds  int
		build   buildRound
		want    []int
		wantErr error
	}{
		{
			name:   "orders results by round when completion order is reversed",
			rounds: 4,
			build:  reverseCompletion(4),
			want:   []int{0, 10, 20, 30},
		},
		{
			name:   "returns the result of a single round",
			rounds: 1,
			build:  reverseCompletion(1),
			want:   []int{0},
		},
		{
			name:    "returns the error of a failing round and cancels the others",
			rounds:  4,
			build:   failingRound(4, 2, errBoom),
			wantErr: errBoom,
		},
		{
			name:    "rejects zero rounds",
			rounds:  0,
			build:   neverCalled,
			wantErr: errNoRounds,
		},
		{
			name:    "rejects negative rounds",
			rounds:  -3,
			build:   neverCalled,
			wantErr: errNoRounds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, verify := tt.build()

			got, err := Collect(context.Background(), tt.rounds, fn)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Collect() error = %v, want %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Collect() = %v, want %v", got, tt.want)
			}
			if verify != nil {
				verify(t)
			}
		})
	}
}

const gateTimeout = 2 * time.Second

type roundFunc = func(ctx context.Context, round int) (int, error)

type buildRound func() (roundFunc, func(*testing.T))

func reverseCompletion(rounds int) buildRound {
	return func() (roundFunc, func(*testing.T)) {
		gates := make([]chan struct{}, rounds)
		for i := range gates {
			gates[i] = make(chan struct{})
		}
		close(gates[rounds-1])

		fn := func(ctx context.Context, round int) (int, error) {
			select {
			case <-gates[round]:
			case <-time.After(gateTimeout):
				return 0, fmt.Errorf("round %d did not run concurrently with the later rounds", round)
			}
			if round > 0 {
				close(gates[round-1])
			}
			return round * 10, nil
		}
		return fn, nil
	}
}

func failingRound(rounds, failAt int, failure error) buildRound {
	return func() (roundFunc, func(*testing.T)) {
		var cancelled atomic.Int64

		fn := func(ctx context.Context, round int) (int, error) {
			if round == failAt {
				return 0, failure
			}
			select {
			case <-ctx.Done():
				cancelled.Add(1)
				return 0, ctx.Err()
			case <-time.After(gateTimeout):
				return 0, fmt.Errorf("round %d was never cancelled", round)
			}
		}
		verify := func(t *testing.T) {
			t.Helper()
			if got, want := cancelled.Load(), int64(rounds-1); got != want {
				t.Errorf("cancelled rounds = %d, want %d", got, want)
			}
		}
		return fn, verify
	}
}

func neverCalled() (roundFunc, func(*testing.T)) {
	var called atomic.Bool

	fn := func(ctx context.Context, round int) (int, error) {
		called.Store(true)
		return 0, nil
	}
	verify := func(t *testing.T) {
		t.Helper()
		if called.Load() {
			t.Error("fn was called for a non-positive round count")
		}
	}
	return fn, verify
}

func TestTally(t *testing.T) {
	tests := []struct {
		name   string
		rounds []map[string]bool
		want   map[string]int
	}{
		{
			name:   "no rounds yields an empty tally",
			rounds: nil,
			want:   map[string]int{},
		},
		{
			name:   "a round without keys contributes nothing",
			rounds: []map[string]bool{{}, {"TestA": true}},
			want:   map[string]int{"TestA": 1},
		},
		{
			name: "every round true counts every round",
			rounds: []map[string]bool{
				{"TestA": true, "TestB": true},
				{"TestA": true, "TestB": true},
			},
			want: map[string]int{"TestA": 2, "TestB": 2},
		},
		{
			name: "every round false counts zero",
			rounds: []map[string]bool{
				{"TestA": false, "TestB": false},
				{"TestA": false, "TestB": false},
			},
			want: map[string]int{"TestA": 0, "TestB": 0},
		},
		{
			name: "disjoint keys across rounds all surface",
			rounds: []map[string]bool{
				{"TestA": true},
				{"TestB": false},
				{"TestC": true},
			},
			want: map[string]int{"TestA": 1, "TestB": 0, "TestC": 1},
		},
		{
			name: "a split vote counts only the rounds that agreed",
			rounds: []map[string]bool{
				{"TestA": true, "TestB": false},
				{"TestA": false, "TestB": false},
				{"TestA": true, "TestB": false},
			},
			want: map[string]int{"TestA": 2, "TestB": 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := cloneRounds(tt.rounds)

			got := Tally(tt.rounds)

			if got == nil {
				t.Errorf("Tally() = nil, want a map so callers can render it as JSON {} rather than null")
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("Tally() = %v, want %v", got, tt.want)
			}
			if !hasSameRounds(tt.rounds, before) {
				t.Errorf("Tally() mutated its input: %v, want %v", tt.rounds, before)
			}
		})
	}
}

func cloneRounds(rounds []map[string]bool) []map[string]bool {
	cloned := make([]map[string]bool, len(rounds))
	for i, round := range rounds {
		cloned[i] = maps.Clone(round)
	}
	return cloned
}

func hasSameRounds(a, b []map[string]bool) bool {
	return slices.EqualFunc(a, b, func(x, y map[string]bool) bool { return maps.Equal(x, y) })
}

func TestIsUnanimous(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		total  int
		want   bool
	}{
		{
			name:   "a nil tally is vacuously unanimous",
			counts: nil,
			total:  4,
			want:   true,
		},
		{
			name:   "an empty tally is vacuously unanimous",
			counts: map[string]int{},
			total:  4,
			want:   true,
		},
		{
			name:   "every count at the total is unanimous",
			counts: map[string]int{"TestA": 4, "TestB": 4},
			total:  4,
			want:   true,
		},
		{
			name:   "a single dissenting count is not unanimous",
			counts: map[string]int{"TestA": 4, "TestB": 3},
			total:  4,
			want:   false,
		},
		{
			name:   "every count at zero is not unanimous",
			counts: map[string]int{"TestA": 0, "TestB": 0},
			total:  4,
			want:   false,
		},
		{
			name:   "a single vote above zero is not unanimous",
			counts: map[string]int{"TestA": 0, "TestB": 1},
			total:  4,
			want:   false,
		},
		{
			name:   "a count above the total is not unanimous",
			counts: map[string]int{"TestA": 5},
			total:  4,
			want:   false,
		},
		{
			name:   "a single vote meeting the total is unanimous",
			counts: map[string]int{"TestA": 1},
			total:  1,
			want:   true,
		},
		{
			name:   "an all-zero tally is unanimous against a total of zero",
			counts: map[string]int{"TestA": 0},
			total:  0,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnanimous(tt.counts, tt.total); got != tt.want {
				t.Errorf("IsUnanimous(%v, %d) = %t, want %t", tt.counts, tt.total, got, tt.want)
			}
		})
	}
}
