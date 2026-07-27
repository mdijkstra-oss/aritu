package vote

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Collect runs fn n times concurrently and returns results in call order. The
// first error cancels the rest and is returned.
func Collect[T any](ctx context.Context, n int, fn func(ctx context.Context, round int) (T, error)) ([]T, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: got %d", errNoRounds, n)
	}

	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]T, n)
	var (
		wg    sync.WaitGroup
		once  sync.Once
		first error
	)
	for round := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := fn(roundCtx, round)
			if err != nil {
				once.Do(func() {
					first = err
					cancel()
				})
				return
			}
			results[round] = result
		}()
	}
	wg.Wait()

	if first != nil {
		return nil, first
	}
	return results, nil
}

// Tally counts, per key, how many rounds voted true. Every key seen in any round
// appears in the result, so a key missing from one round still surfaces.
func Tally[K comparable](rounds []map[K]bool) map[K]int {
	counts := make(map[K]int)
	for _, round := range rounds {
		for key, isSatisfied := range round {
			count := counts[key]
			if isSatisfied {
				count++
			}
			counts[key] = count
		}
	}
	return counts
}

// IsUnanimous reports whether every count equals total. An empty tally is
// unanimous: a file with no tests has nothing to fail.
func IsUnanimous[K comparable](counts map[K]int, total int) bool {
	for _, count := range counts {
		if count != total {
			return false
		}
	}
	return true
}

var errNoRounds = errors.New("vote: rounds must be positive")
